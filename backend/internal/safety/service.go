package safety

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ledger_two/internal/config"
	"ledger_two/internal/errors"
	ledgerctx "ledger_two/internal/ledger"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
)

type Service struct {
	db             *sql.DB
	cfg            *config.Config
	instancePolicy ledgerctx.InstancePolicy
}

func NewService(db *sql.DB, cfg *config.Config) *Service {
	return &Service{
		db:             db,
		cfg:            cfg,
		instancePolicy: ledgerctx.NewInstancePolicy(ledgerctx.NewRepository(db)),
	}
}

type BackupInfo struct {
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type RestorePreparation struct {
	Filename         string `json:"filename"`
	Instructions     string `json:"instructions"`
	RequiresDowntime bool   `json:"requires_downtime"`
}

type BackupDownload struct {
	File       *os.File
	Filename   string
	SizeBytes  int64
	ModifiedAt time.Time
}

type Diagnostics struct {
	Env               string             `json:"env"`
	DeploymentChannel string             `json:"deployment_channel"`
	AppBaseURLSet     bool               `json:"app_base_url_set"`
	CookieSecure      string             `json:"cookie_secure"`
	CookieSameSite    string             `json:"cookie_samesite"`
	Database          DiagnosticStatus   `json:"database"`
	Storage           []DiagnosticStatus `json:"storage"`
	LatestBackup      *BackupInfo        `json:"latest_backup,omitempty"`
	AuditActionCount  map[string]int     `json:"audit_action_count"`
	GeneratedAt       time.Time          `json:"generated_at"`
}

type DiagnosticStatus struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	Configured bool   `json:"configured"`
	Writable   *bool  `json:"writable,omitempty"`
	Message    string `json:"message,omitempty"`
	Version    int64  `json:"version,omitempty"`
}

// checkBackupDirWritable 验证备份目录是否可写
func (s *Service) checkBackupDirWritable(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupPathInvalid, "failed to create backup directory: "+err.Error())
	}
	testFile := filepath.Join(dir, fmt.Sprintf(".write_test_%d", time.Now().UnixNano()))
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupPathInvalid, "backup directory is not writable: "+err.Error())
	}
	_ = os.Remove(testFile)
	return nil
}

func (s *Service) requireInstanceAdmin(ctx context.Context, userID string) error {
	allowed, err := s.instancePolicy.Can(ctx, userID)
	if err != nil {
		return errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "实例权限校验失败")
	}
	if !allowed {
		return errors.NewAppError(http.StatusForbidden, errors.ErrCodeInstanceAdminRequired, "需要实例管理员权限")
	}
	return nil
}

func (s *Service) Diagnostics(ctx context.Context, actorUserID string) (*Diagnostics, error) {
	if err := s.requireInstanceAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	diag := &Diagnostics{
		Env:               "unknown",
		DeploymentChannel: "unknown",
		CookieSameSite:    "Lax",
		Database:          s.databaseStatus(ctx),
		Storage:           make([]DiagnosticStatus, 0, 4),
		AuditActionCount:  make(map[string]int),
		GeneratedAt:       time.Now(),
	}
	if s.cfg != nil {
		diag.Env = emptyAsUnknown(s.cfg.Env)
		diag.DeploymentChannel = emptyAsUnknown(s.cfg.DeploymentChannel)
		diag.AppBaseURLSet = strings.TrimSpace(s.cfg.AppBaseURL) != ""
		diag.CookieSecure = emptyAsUnknown(s.cfg.CookieSecure)
		diag.CookieSameSite = emptyAsUnknown(s.cfg.CookieSameSite)
		diag.Storage = append(diag.Storage,
			s.storageStatus("database_dir", "数据库目录", sqliteParentDir(s.cfg.DSN), true),
			s.storageStatus("backup_dir", "备份目录", s.cfg.BackupDir, true),
			s.storageStatus("upload_dir", "上传目录", s.cfg.UploadDir, true),
			s.storageStatus("log_dir", "日志目录", s.cfg.LogDir, true),
		)
	} else {
		diag.Storage = append(diag.Storage,
			statusNotConfigured("database_dir", "数据库目录"),
			statusNotConfigured("backup_dir", "备份目录"),
			statusNotConfigured("upload_dir", "上传目录"),
			statusNotConfigured("log_dir", "日志目录"),
		)
	}

	if backups, err := s.scanBackups(); err == nil && len(backups) > 0 {
		diag.LatestBackup = &backups[0]
	}
	diag.AuditActionCount = s.auditActionCounts(ctx)
	afterBytes, _ := json.Marshal(map[string]interface{}{
		"deployment_channel": diag.DeploymentChannel,
		"database_status":    diag.Database.Status,
		"schema_version":     diag.Database.Version,
		"storage_items":      len(diag.Storage),
	})
	if err := s.recordInstanceAudit(
		ctx,
		actorUserID,
		"system_diagnostics",
		"instance",
		"current",
		string(afterBytes),
		time.Now().UTC(),
	); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "记录系统诊断审计失败")
	}
	return diag, nil
}

func (s *Service) databaseStatus(ctx context.Context) DiagnosticStatus {
	status := DiagnosticStatus{
		Key:        "sqlite",
		Label:      "SQLite 数据库",
		Status:     "ok",
		Configured: s.db != nil,
	}
	if s.db == nil {
		status.Status = "error"
		status.Message = "database connection is not configured"
		return status
	}
	if err := s.db.PingContext(ctx); err != nil {
		status.Status = "error"
		status.Message = "database ping failed"
		return status
	}
	version, err := goose.GetDBVersion(s.db)
	if err != nil {
		status.Status = "warning"
		status.Message = "schema version unavailable"
		return status
	}
	status.Version = version
	return status
}

func (s *Service) storageStatus(key string, label string, dir string, shouldWrite bool) DiagnosticStatus {
	if strings.TrimSpace(dir) == "" {
		return statusNotConfigured(key, label)
	}
	ok := true
	status := DiagnosticStatus{
		Key:        key,
		Label:      label,
		Status:     "ok",
		Configured: true,
		Writable:   &ok,
	}
	if !shouldWrite {
		return status
	}
	if err := ensureDiagnosticWritableDir(dir); err != nil {
		ok = false
		status.Writable = &ok
		status.Status = "error"
		status.Message = "directory is not writable"
	}
	return status
}

func statusNotConfigured(key string, label string) DiagnosticStatus {
	ok := false
	return DiagnosticStatus{
		Key:        key,
		Label:      label,
		Status:     "warning",
		Configured: false,
		Writable:   &ok,
		Message:    "directory is not configured",
	}
}

func ensureDiagnosticWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	testFile, err := os.CreateTemp(dir, ".diagnostic_write_test_*")
	if err != nil {
		return err
	}
	testName := testFile.Name()
	if err := testFile.Close(); err != nil {
		_ = os.Remove(testName)
		return err
	}
	return os.Remove(testName)
}

func sqliteParentDir(dsn string) string {
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return ""
	}
	path := dsn
	if strings.HasPrefix(path, "file:") {
		path = strings.TrimPrefix(path, "file:")
		if idx := strings.Index(path, "?"); idx >= 0 {
			path = path[:idx]
		}
	}
	dir := filepath.Dir(path)
	if dir == "." {
		return ""
	}
	return dir
}

func emptyAsUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func (s *Service) auditActionCounts(ctx context.Context) map[string]int {
	counts := make(map[string]int)
	if s.db == nil {
		return counts
	}
	rows, err := s.db.QueryContext(ctx, "SELECT action, COUNT(*) FROM instance_audit_logs GROUP BY action")
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err == nil {
			counts[action] = count
		}
	}
	return counts
}

func (s *Service) recordInstanceAudit(ctx context.Context, actorUserID string, action string, entityType string, entityID string, afterJSON string, createdAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO instance_audit_logs (
			id, actor_user_id, action, entity_type, entity_id,
			before_json, after_json, created_at
		) VALUES (?, ?, ?, ?, ?, NULL, ?, ?)
	`, uuid.NewString(), actorUserID, action, entityType, entityID, afterJSON, createdAt.Format(time.RFC3339))
	return err
}

// ManualBackup 手动备份 SQLite 数据库
func (s *Service) ManualBackup(ctx context.Context, actorUserID string) (*BackupInfo, error) {
	if err := s.requireInstanceAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	backupDir, err := s.backupRoot()
	if err != nil {
		return nil, err
	}
	manualDir := filepath.Join(backupDir, "manual")
	if err := s.checkBackupDirWritable(manualDir); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	// 避免文件名中包含空格或冒号，采用简洁的时间格式
	filename := fmt.Sprintf("backup_manual_%s_%d.db", now.Format("20060102_150405"), now.Nanosecond())
	backupPath := filepath.Join(manualDir, filename)

	// 使用 VACUUM INTO 进行安全在线备份，防止热拷贝锁死
	safePath := strings.ReplaceAll(backupPath, "'", "''")
	query := fmt.Sprintf("VACUUM INTO '%s'", safePath)
	_, err = s.db.ExecContext(ctx, query)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "database vacuum into failed: "+err.Error())
	}

	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		_ = os.Remove(backupPath)
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "读取备份文件元数据失败")
	}
	backup := &BackupInfo{
		Filename:  "manual/" + filename,
		SizeBytes: fileInfo.Size(),
		CreatedAt: fileInfo.ModTime().UTC(),
	}

	// 记录实例级审计日志，不伪造账本 ID。
	afterJSONObj := map[string]interface{}{
		"filename":   backup.Filename,
		"size_bytes": backup.SizeBytes,
	}
	afterBytes, _ := json.Marshal(afterJSONObj)

	if err := s.recordInstanceAudit(ctx, actorUserID, "manual_database_backup", "database", backup.Filename, string(afterBytes), now); err != nil {
		_ = os.Remove(backupPath)
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "failed to record backup audit log: "+err.Error())
	}

	return backup, nil
}

// RestoreBackup 准备恢复流程。因操作系统文件锁机制，后端仅执行自动前置备份并返回操作指引。
func (s *Service) RestoreBackup(ctx context.Context, actorUserID string, targetFilename string) (*RestorePreparation, error) {
	if err := s.requireInstanceAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	backupDir, err := s.backupRoot()
	if err != nil {
		return nil, err
	}
	targetKey, _, _, err := s.resolveBackupFile(targetFilename)
	if err != nil {
		return nil, err
	}

	// 1. 强制执行前置备份 (后悔药)
	now := time.Now().UTC()
	manualDir := filepath.Join(backupDir, "manual")
	if err := s.checkBackupDirWritable(manualDir); err != nil {
		return nil, err
	}

	preFilename := fmt.Sprintf("pre_restore_%s_%d.db", now.Format("20060102_150405"), now.Nanosecond())
	preBackupPath := filepath.Join(manualDir, preFilename)

	safePath := strings.ReplaceAll(preBackupPath, "'", "''")
	query := fmt.Sprintf("VACUUM INTO '%s'", safePath)
	_, err = s.db.ExecContext(ctx, query)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "自动前置备份失败，终止恢复流程: "+err.Error())
	}

	afterJSONObj := map[string]interface{}{
		"action":          "pre_restore_backup",
		"target_restore":  targetKey,
		"pre_backup_file": "manual/" + preFilename,
	}
	afterBytes, _ := json.Marshal(afterJSONObj)
	if err := s.recordInstanceAudit(ctx, actorUserID, "prepare_database_restore", "database", targetKey, string(afterBytes), now); err != nil {
		_ = os.Remove(preBackupPath)
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "记录恢复准备审计失败")
	}

	// 2. 返回人工覆盖指引（策略 B）
	instructions := fmt.Sprintf("已自动为您创建当前数据的安全备份 (%s)。为避免数据损坏，请关闭程序后，手动将文件 %s 覆盖至 data/ledger.db，随后重新启动服务即可生效。", preFilename, targetKey)

	return &RestorePreparation{
		Filename:         targetKey,
		Instructions:     instructions,
		RequiresDowntime: true,
	}, nil
}

// ListBackups 获取所有备份文件列表，并记录实例级读取审计。
func (s *Service) ListBackups(ctx context.Context, actorUserID string) ([]BackupInfo, error) {
	if err := s.requireInstanceAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}
	list, err := s.scanBackups()
	if err != nil {
		return nil, err
	}
	afterBytes, _ := json.Marshal(map[string]interface{}{"records": len(list)})
	if err := s.recordInstanceAudit(
		ctx,
		actorUserID,
		"list_database_backups",
		"database_backup",
		"all",
		string(afterBytes),
		time.Now().UTC(),
	); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "记录备份列表审计失败")
	}
	return list, nil
}

// OpenBackupDownload 打开一个受管理的备份文件，并在返回文件流前记录实例审计。
func (s *Service) OpenBackupDownload(ctx context.Context, actorUserID, filename string) (*BackupDownload, error) {
	if err := s.requireInstanceAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}
	key, targetPath, fileInfo, err := s.resolveBackupFile(filename)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(targetPath)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "打开备份文件失败")
	}
	afterBytes, _ := json.Marshal(map[string]interface{}{
		"filename":   key,
		"size_bytes": fileInfo.Size(),
	})
	if err := s.recordInstanceAudit(
		ctx,
		actorUserID,
		"download_database_backup",
		"database_backup",
		key,
		string(afterBytes),
		time.Now().UTC(),
	); err != nil {
		_ = file.Close()
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "记录备份下载审计失败")
	}
	return &BackupDownload{
		File:       file,
		Filename:   key,
		SizeBytes:  fileInfo.Size(),
		ModifiedAt: fileInfo.ModTime(),
	}, nil
}

func (s *Service) scanBackups() ([]BackupInfo, error) {
	backupDir, err := s.backupRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupPathInvalid, "备份目录不可用")
	}

	list := make([]BackupInfo, 0)
	err = filepath.Walk(backupDir, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info == nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(info.Name()), ".db") {
			return nil
		}

		rel, relErr := filepath.Rel(backupDir, filePath)
		if relErr != nil {
			return relErr
		}
		key := filepath.ToSlash(rel)
		if _, err := normalizeBackupKey(key); err != nil {
			return nil
		}

		list = append(list, BackupInfo{
			Filename:  key,
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "扫描备份目录失败")
	}

	// 降序排序（最新修改的备份在前），同一时间按 key 保持确定性。
	sort.Slice(list, func(i, j int) bool {
		if list[i].CreatedAt.Equal(list[j].CreatedAt) {
			return list[i].Filename < list[j].Filename
		}
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	return list, nil
}

func (s *Service) backupRoot() (string, error) {
	if s.cfg == nil || strings.TrimSpace(s.cfg.BackupDir) == "" {
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupPathInvalid, "备份目录未配置")
	}
	root, err := filepath.Abs(filepath.Clean(s.cfg.BackupDir))
	if err != nil {
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupPathInvalid, "备份目录无效")
	}
	return root, nil
}

func (s *Service) resolveBackupFile(filename string) (string, string, os.FileInfo, error) {
	key, err := normalizeBackupKey(filename)
	if err != nil {
		return "", "", nil, err
	}
	root, err := s.backupRoot()
	if err != nil {
		return "", "", nil, err
	}
	targetPath := filepath.Join(root, filepath.FromSlash(key))
	rel, err := filepath.Rel(root, targetPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", nil, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份文件名无效")
	}

	currentPath := root
	segments := strings.Split(key, "/")
	var fileInfo os.FileInfo
	for index, segment := range segments {
		currentPath = filepath.Join(currentPath, segment)
		fileInfo, err = os.Lstat(currentPath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", "", nil, errors.NewAppError(http.StatusNotFound, errors.ErrCodeBackupNotFound, "备份文件不存在")
			}
			return "", "", nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "读取备份文件元数据失败")
		}
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return "", "", nil, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份路径不能包含符号链接")
		}
		if index < len(segments)-1 && !fileInfo.IsDir() {
			return "", "", nil, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份文件名无效")
		}
	}
	if fileInfo == nil || fileInfo.IsDir() {
		return "", "", nil, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份文件名无效")
	}
	return key, targetPath, fileInfo, nil
}

func normalizeBackupKey(value string) (string, error) {
	key := strings.TrimSpace(value)
	if key == "" || strings.Contains(key, "\\") || path.Clean(key) != key || strings.HasPrefix(key, "/") {
		return "", errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份文件名无效")
	}
	if !strings.EqualFold(path.Ext(key), ".db") {
		return "", errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份文件必须是 .db 文件")
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份文件名无效")
		}
		for _, char := range segment {
			if (char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '.' || char == '_' || char == '-' {
				continue
			}
			return "", errors.NewAppError(http.StatusBadRequest, errors.ErrCodeValidationError, "备份文件名包含不允许的字符")
		}
	}
	return key, nil
}

// ExportCSV 导出当前登录用户有权查看的交易流水为 CSV 内容
func (s *Service) ExportCSV(ctx context.Context, actorUserID string, month string) ([]byte, error) {
	lc, err := ledgerctx.RequireExplicitLedgerContext(ctx, actorUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeLedgerRequired, "请选择账本后再导出")
	}
	ledgerID := lc.LedgerID

	// 1. 查询用户
	userMap := make(map[string]string) // id -> display_name
	uRows, err := s.db.QueryContext(ctx, `
		WITH visible_transactions AS (
			SELECT owner_user_id, created_by_user_id, payer_user_id
			FROM transactions
			WHERE ledger_id = ?
			  AND status != 'deleted'
			  AND (
				created_by_user_id = ?
				OR owner_user_id = ?
				OR payer_user_id = ?
				OR visibility IN ('partner_readable', 'shared')
			  )
		),
		relevant_user_ids AS (
			SELECT user_id FROM ledger_members WHERE ledger_id = ?
			UNION
			SELECT owner_user_id FROM visible_transactions
			UNION
			SELECT created_by_user_id FROM visible_transactions
			UNION
			SELECT payer_user_id FROM visible_transactions WHERE payer_user_id IS NOT NULL
		)
		SELECT users.id, users.display_name, users.username
		FROM relevant_user_ids relevant
		JOIN users ON users.id = relevant.user_id
		ORDER BY users.id ASC
	`, ledgerID, actorUserID, actorUserID, actorUserID, ledgerID)
	if err == nil {
		defer uRows.Close()
		for uRows.Next() {
			var id, display, name string
			if err := uRows.Scan(&id, &display, &name); err == nil {
				if display != "" {
					userMap[id] = display
				} else {
					userMap[id] = name
				}
			}
		}
	}

	// 2. 查询分类
	catMap := make(map[string]string) // id -> name
	cRows, err := s.db.QueryContext(ctx, "SELECT id, name FROM categories WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var id, name string
			if err := cRows.Scan(&id, &name); err == nil {
				catMap[id] = name
			}
		}
	}

	// 3. 查询标签映射
	tagMap := make(map[string][]string) // txID -> tagNames
	tRows, err := s.db.QueryContext(ctx, `
		SELECT tt.transaction_id, t.name 
		FROM transaction_tags tt
		JOIN tags t ON tt.tag_id = t.id
		JOIN transactions parent ON parent.id = tt.transaction_id
		WHERE t.ledger_id = ? AND parent.ledger_id = ?
	`, ledgerID, ledgerID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var txID, name string
			if err := tRows.Scan(&txID, &name); err == nil {
				tagMap[txID] = append(tagMap[txID], name)
			}
		}
	}

	// 4. 查询当前用户可见的 transactions (排除 deleted)
	query := `
		SELECT 
			id, type, title, amount, currency, occurred_at, 
			owner_user_id, created_by_user_id, payer_user_id, category_id, visibility, note
		FROM transactions
		WHERE ledger_id = ? AND status != 'deleted'
		AND (
			created_by_user_id = ? 
			OR owner_user_id = ? 
			OR payer_user_id = ? 
			OR visibility IN ('partner_readable', 'shared')
		)
	`
	args := []interface{}{ledgerID, actorUserID, actorUserID, actorUserID}
	if month != "" {
		query += " AND occurred_at LIKE ?"
		args = append(args, month+"%")
	}
	query += " ORDER BY occurred_at DESC, created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeExportFailed, "failed to query transactions for export: "+err.Error())
	}
	defer rows.Close()

	var sb strings.Builder
	// 写入 UTF-8 BOM
	sb.Write([]byte{0xEF, 0xBB, 0xBF})
	// 写入 CSV 头部
	sb.WriteString("发生时间,类型,标题,金额分,金额元,分类,标签,付款人,归属人,可见性,备注\n")

	txCount := 0
	for rows.Next() {
		var id, txtype, title, currency, occurredAt, ownerID, createdBy, payerID, visibility string
		var amount int64
		var catIDOpt, noteOpt sql.NullString

		err := rows.Scan(&id, &txtype, &title, &amount, &currency, &occurredAt,
			&ownerID, &createdBy, &payerID, &catIDOpt, &visibility, &noteOpt)
		if err != nil {
			return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeExportFailed, "scan transaction error: "+err.Error())
		}

		// 格式化时间为友好字符串，如果包含 'T' 可以适当转换
		tStr := occurredAt
		if parsedTime, err := time.Parse(time.RFC3339, occurredAt); err == nil {
			tStr = parsedTime.Format("2006-01-02 15:04:05")
		}

		// 翻译类型与可见性以便于阅读
		typeCN := txtype
		switch txtype {
		case "expense":
			typeCN = "支出"
		case "income":
			typeCN = "收入"
		case "settlement":
			typeCN = "结算"
		}

		visibilityCN := visibility
		switch visibility {
		case "private":
			visibilityCN = "私有"
		case "partner_readable":
			visibilityCN = "对方可见"
		case "shared":
			visibilityCN = "完全共享"
		}

		amountYuan := fmt.Sprintf("%.2f", float64(amount)/100.0)

		catName := ""
		if catIDOpt.Valid {
			catName = catMap[catIDOpt.String]
		}

		tagsStr := ""
		if tags, exists := tagMap[id]; exists {
			tagsStr = strings.Join(tags, " ")
		}

		payerName := userMap[payerID]
		ownerName := userMap[ownerID]

		note := ""
		if noteOpt.Valid {
			note = noteOpt.String
		}

		// 转义 CSV 中的逗号与换行符
		csvTitle := escapeCSVField(title)
		csvNote := escapeCSVField(note)
		csvCatName := escapeCSVField(catName)
		csvTagsStr := escapeCSVField(tagsStr)

		rowLine := fmt.Sprintf("%s,%s,%s,%d,%s,%s,%s,%s,%s,%s,%s\n",
			tStr, typeCN, csvTitle, amount, amountYuan, csvCatName, csvTagsStr, payerName, ownerName, visibilityCN, csvNote)
		sb.WriteString(rowLine)
		txCount++
	}

	// 记录审计日志
	afterObj := map[string]interface{}{
		"export_type": "csv",
		"month":       month,
		"records":     txCount,
	}
	afterBytes, _ := json.Marshal(afterObj)

	auditQuery := `
		INSERT INTO audit_logs (id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at)
		VALUES (?, ?, ?, 'export_csv', 'transaction', 'csv', NULL, ?, ?)
	`
	_, _ = s.db.ExecContext(ctx, auditQuery,
		uuid.NewString(), ledgerID, actorUserID, string(afterBytes), time.Now().Format(time.RFC3339))

	return []byte(sb.String()), nil
}

// escapeCSVField 对 CSV 字段中的特殊字符（逗号、双引号、换行符）进行安全转义
func escapeCSVField(f string) string {
	if strings.ContainsAny(f, ",\"\n\r") {
		return `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
	}
	return f
}

// ExportJSON 导出当前用户可见的所有数据块并进行脱敏
func (s *Service) ExportJSON(ctx context.Context, actorUserID string) ([]byte, error) {
	lc, err := ledgerctx.RequireExplicitLedgerContext(ctx, actorUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, errors.ErrCodeLedgerRequired, "请选择账本后再导出")
	}
	ledgerID := lc.LedgerID

	// 1. 导出脱敏的 users
	type UserDTO struct {
		ID          string  `json:"id"`
		Username    string  `json:"username"`
		DisplayName string  `json:"display_name"`
		AvatarURL   *string `json:"avatar_url,omitempty"`
		Role        string  `json:"role"`
		IsActive    int     `json:"is_active"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}
	users := make([]UserDTO, 0)
	uRows, err := s.db.QueryContext(ctx, `
		WITH visible_transactions AS (
			SELECT id, owner_user_id, created_by_user_id, payer_user_id
			FROM transactions
			WHERE ledger_id = ?
			  AND status != 'deleted'
			  AND (
				created_by_user_id = ?
				OR owner_user_id = ?
				OR payer_user_id = ?
				OR visibility IN ('partner_readable', 'shared')
			  )
		),
		relevant_user_ids AS (
			SELECT user_id
			FROM ledger_members
			WHERE ledger_id = ?
			UNION
			SELECT owner_user_id FROM visible_transactions
			UNION
			SELECT created_by_user_id FROM visible_transactions
			UNION
			SELECT payer_user_id FROM visible_transactions WHERE payer_user_id IS NOT NULL
			UNION
			SELECT splits.user_id
			FROM transaction_splits splits
			JOIN visible_transactions visible ON visible.id = splits.transaction_id
			UNION
			SELECT from_user_id FROM settlements WHERE ledger_id = ?
			UNION
			SELECT to_user_id FROM settlements WHERE ledger_id = ?
			UNION
			SELECT created_by_user_id FROM settlements WHERE ledger_id = ?
			UNION
			SELECT created_by_user_id FROM transaction_templates WHERE ledger_id = ?
			UNION
			SELECT created_by_user_id FROM recurring_rules WHERE ledger_id = ?
			UNION
			SELECT created_by_user_id FROM import_batches WHERE ledger_id = ?
			UNION
			SELECT created_by_user_id FROM import_rules WHERE ledger_id = ?
			UNION
			SELECT actor_user_id
			FROM audit_logs
			WHERE ledger_id = ?
			  AND (
				entity_type != 'transaction'
				OR entity_id IN (SELECT id FROM visible_transactions)
			  )
		)
		SELECT users.id, users.username, users.display_name, users.avatar_url,
		       COALESCE(ledger_members.role, 'historical'),
		       users.is_active, users.created_at, users.updated_at
		FROM relevant_user_ids relevant
		JOIN users ON users.id = relevant.user_id
		LEFT JOIN ledger_members
		  ON ledger_members.ledger_id = ?
		 AND ledger_members.user_id = users.id
		ORDER BY
			CASE WHEN ledger_members.user_id IS NULL THEN 1 ELSE 0 END,
			ledger_members.created_at ASC,
			users.id ASC
	`,
		ledgerID,
		actorUserID,
		actorUserID,
		actorUserID,
		ledgerID,
		ledgerID,
		ledgerID,
		ledgerID,
		ledgerID,
		ledgerID,
		ledgerID,
		ledgerID,
		ledgerID,
		ledgerID,
	)
	if err == nil {
		defer uRows.Close()
		for uRows.Next() {
			var u UserDTO
			var avatar sql.NullString
			if err := uRows.Scan(&u.ID, &u.Username, &u.DisplayName, &avatar, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err == nil {
				if avatar.Valid {
					u.AvatarURL = &avatar.String
				}
				users = append(users, u)
			}
		}
	}

	// 2. 导出 categories
	type CategoryDTO struct {
		ID          string  `json:"id"`
		LedgerID    string  `json:"ledger_id"`
		OwnerUserID *string `json:"owner_user_id,omitempty"`
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Icon        *string `json:"icon,omitempty"`
		Color       *string `json:"color,omitempty"`
		ParentID    *string `json:"parent_id,omitempty"`
		SortOrder   int     `json:"sort_order"`
		IsSystem    int     `json:"is_system"`
		IsArchived  int     `json:"is_archived"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}
	categories := make([]CategoryDTO, 0)
	cRows, err := s.db.QueryContext(ctx, "SELECT id, ledger_id, owner_user_id, name, type, icon, color, parent_id, sort_order, is_system, is_archived, created_at, updated_at FROM categories WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var c CategoryDTO
			var owner, icon, color, parent sql.NullString
			err := cRows.Scan(&c.ID, &c.LedgerID, &owner, &c.Name, &c.Type, &icon, &color, &parent, &c.SortOrder, &c.IsSystem, &c.IsArchived, &c.CreatedAt, &c.UpdatedAt)
			if err == nil {
				if owner.Valid {
					c.OwnerUserID = &owner.String
				}
				if icon.Valid {
					c.Icon = &icon.String
				}
				if color.Valid {
					c.Color = &color.String
				}
				if parent.Valid {
					c.ParentID = &parent.String
				}
				categories = append(categories, c)
			}
		}
	}

	// 3. 导出 tags
	type TagDTO struct {
		ID          string  `json:"id"`
		LedgerID    string  `json:"ledger_id"`
		Name        string  `json:"name"`
		OwnerUserID *string `json:"owner_user_id,omitempty"`
		Color       *string `json:"color,omitempty"`
		SortOrder   int     `json:"sort_order"`
		IsArchived  int     `json:"is_archived"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}
	tags := make([]TagDTO, 0)
	tRows, err := s.db.QueryContext(ctx, "SELECT id, ledger_id, name, owner_user_id, color, sort_order, is_archived, created_at, updated_at FROM tags WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var t TagDTO
			var owner, color sql.NullString
			if err := tRows.Scan(&t.ID, &t.LedgerID, &t.Name, &owner, &color, &t.SortOrder, &t.IsArchived, &t.CreatedAt, &t.UpdatedAt); err == nil {
				if owner.Valid {
					t.OwnerUserID = &owner.String
				}
				if color.Valid {
					t.Color = &color.String
				}
				tags = append(tags, t)
			}
		}
	}

	// 4. 导出 accounts
	type AccountDTO struct {
		ID             string `json:"id"`
		LedgerID       string `json:"ledger_id"`
		OwnerUserID    string `json:"owner_user_id"`
		Name           string `json:"name"`
		Type           string `json:"type"`
		Currency       string `json:"currency"`
		InitialBalance int64  `json:"initial_balance"`
		SortOrder      int    `json:"sort_order"`
		IsArchived     int    `json:"is_archived"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	}
	accounts := make([]AccountDTO, 0)
	aRows, err := s.db.QueryContext(ctx, "SELECT id, ledger_id, owner_user_id, name, type, currency, initial_balance, sort_order, is_archived, created_at, updated_at FROM accounts WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a AccountDTO
			if err := aRows.Scan(&a.ID, &a.LedgerID, &a.OwnerUserID, &a.Name, &a.Type, &a.Currency, &a.InitialBalance, &a.SortOrder, &a.IsArchived, &a.CreatedAt, &a.UpdatedAt); err == nil {
				accounts = append(accounts, a)
			}
		}
	}

	// 5. 查询当前用户可见的 transactions (排除 status = 'deleted')
	type TransactionDTO struct {
		ID              string   `json:"id"`
		LedgerID        string   `json:"ledger_id"`
		Type            string   `json:"type"`
		Title           string   `json:"title"`
		Amount          int64    `json:"amount"`
		Currency        string   `json:"currency"`
		OccurredAt      string   `json:"occurred_at"`
		OwnerUserID     string   `json:"owner_user_id"`
		CreatedByUserID string   `json:"created_by_user_id"`
		PayerUserID     *string  `json:"payer_user_id,omitempty"`
		AccountID       *string  `json:"account_id,omitempty"`
		CategoryID      *string  `json:"category_id,omitempty"`
		Visibility      string   `json:"visibility"`
		SplitMethod     *string  `json:"split_method,omitempty"`
		Note            *string  `json:"note,omitempty"`
		AttachmentPaths []string `json:"attachment_paths"`
		Status          string   `json:"status"`
		CreatedAt       string   `json:"created_at"`
		UpdatedAt       string   `json:"updated_at"`
	}
	transactions := make([]TransactionDTO, 0)
	visibleTxIDs := make(map[string]bool)

	txQuery := `
		SELECT 
			id, ledger_id, type, title, amount, currency, occurred_at, 
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id, 
			visibility, split_method, note, attachment_paths, status, created_at, updated_at
		FROM transactions 
		WHERE ledger_id = ? AND status != 'deleted' 
		AND (
			created_by_user_id = ? 
			OR owner_user_id = ? 
			OR payer_user_id = ? 
			OR visibility IN ('partner_readable', 'shared')
		)
	`
	txRows, err := s.db.QueryContext(ctx, txQuery, ledgerID, actorUserID, actorUserID, actorUserID)
	if err == nil {
		defer txRows.Close()
		for txRows.Next() {
			var tx TransactionDTO
			var payer, account, category, split, note, attachments sql.NullString
			err := txRows.Scan(&tx.ID, &tx.LedgerID, &tx.Type, &tx.Title, &tx.Amount, &tx.Currency, &tx.OccurredAt,
				&tx.OwnerUserID, &tx.CreatedByUserID, &payer, &account, &category,
				&tx.Visibility, &split, &note, &attachments, &tx.Status, &tx.CreatedAt, &tx.UpdatedAt)
			if err == nil {
				if payer.Valid {
					tx.PayerUserID = &payer.String
				}
				if account.Valid {
					tx.AccountID = &account.String
				}
				if category.Valid {
					tx.CategoryID = &category.String
				}
				if split.Valid {
					tx.SplitMethod = &split.String
				}
				if note.Valid {
					tx.Note = &note.String
				}
				tx.AttachmentPaths = make([]string, 0)
				if attachments.Valid && attachments.String != "" {
					_ = json.Unmarshal([]byte(attachments.String), &tx.AttachmentPaths)
				}
				transactions = append(transactions, tx)
				visibleTxIDs[tx.ID] = true
			}
		}
	}

	// 6. 导出 transaction_splits (只包含可见交易对应的 splits)
	type SplitDTO struct {
		ID            string `json:"id"`
		TransactionID string `json:"transaction_id"`
		UserID        string `json:"user_id"`
		ShareAmount   int64  `json:"share_amount"`
		ShareRatio    *int   `json:"share_ratio,omitempty"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
	}
	splits := make([]SplitDTO, 0)
	sRows, err := s.db.QueryContext(ctx, `
		SELECT splits.id, splits.transaction_id, splits.user_id, splits.share_amount,
		       splits.share_ratio, splits.created_at, splits.updated_at
		FROM transaction_splits splits
		JOIN transactions parent ON parent.id = splits.transaction_id
		WHERE parent.ledger_id = ?
	`, ledgerID)
	if err == nil {
		defer sRows.Close()
		for sRows.Next() {
			var s SplitDTO
			var ratio sql.NullInt32
			if err := sRows.Scan(&s.ID, &s.TransactionID, &s.UserID, &s.ShareAmount, &ratio, &s.CreatedAt, &s.UpdatedAt); err == nil {
				if visibleTxIDs[s.TransactionID] {
					if ratio.Valid {
						r := int(ratio.Int32)
						s.ShareRatio = &r
					}
					splits = append(splits, s)
				}
			}
		}
	}

	// 7. 导出 settlements
	type SettlementDTO struct {
		ID              string  `json:"id"`
		LedgerID        string  `json:"ledger_id"`
		FromUserID      string  `json:"from_user_id"`
		ToUserID        string  `json:"to_user_id"`
		Amount          int64   `json:"amount"`
		Currency        string  `json:"currency"`
		OccurredAt      string  `json:"occurred_at"`
		Note            *string `json:"note,omitempty"`
		CreatedByUserID string  `json:"created_by_user_id"`
		CreatedAt       string  `json:"created_at"`
	}
	settlements := make([]SettlementDTO, 0)
	setRows, err := s.db.QueryContext(ctx, "SELECT id, ledger_id, from_user_id, to_user_id, amount, currency, occurred_at, note, created_by_user_id, created_at FROM settlements WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer setRows.Close()
		for setRows.Next() {
			var set SettlementDTO
			var note sql.NullString
			if err := setRows.Scan(&set.ID, &set.LedgerID, &set.FromUserID, &set.ToUserID, &set.Amount, &set.Currency, &set.OccurredAt, &note, &set.CreatedByUserID, &set.CreatedAt); err == nil {
				if note.Valid {
					set.Note = &note.String
				}
				settlements = append(settlements, set)
			}
		}
	}

	// 8. 导出 audit_logs (过滤不可见账单相关的记录)
	type AuditLogDTO struct {
		ID          string  `json:"id"`
		LedgerID    string  `json:"ledger_id"`
		ActorUserID string  `json:"actor_user_id"`
		ActorRole   *string `json:"actor_role,omitempty"`
		Action      string  `json:"action"`
		EntityType  string  `json:"entity_type"`
		EntityID    string  `json:"entity_id"`
		BeforeJSON  *string `json:"before_json,omitempty"`
		AfterJSON   *string `json:"after_json,omitempty"`
		CreatedAt   string  `json:"created_at"`
	}
	auditLogs := make([]AuditLogDTO, 0)
	aRows, err = s.db.QueryContext(ctx, "SELECT id, ledger_id, actor_user_id, actor_role, action, entity_type, entity_id, before_json, after_json, created_at FROM audit_logs WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a AuditLogDTO
			var actorRole, before, after sql.NullString
			err := aRows.Scan(&a.ID, &a.LedgerID, &a.ActorUserID, &actorRole, &a.Action, &a.EntityType, &a.EntityID, &before, &after, &a.CreatedAt)
			if err == nil {
				// 权限收拢过滤：若为交易实体审计，且该交易对当前登录人不可见，则安全排除该条审计行
				if a.EntityType == "transaction" && !visibleTxIDs[a.EntityID] {
					continue
				}
				if before.Valid {
					a.BeforeJSON = &before.String
				}
				if actorRole.Valid {
					a.ActorRole = &actorRole.String
				}
				if after.Valid {
					a.AfterJSON = &after.String
				}
				auditLogs = append(auditLogs, a)
			}
		}
	}

	loadExportSection := func(section string, query string, args ...any) ([]map[string]any, error) {
		rows, queryErr := queryLedgerExportSection(ctx, s.db, section, query, args...)
		if queryErr != nil {
			return nil, errors.NewAppError(
				http.StatusInternalServerError,
				errors.ErrCodeExportFailed,
				queryErr.Error(),
			)
		}
		return rows, nil
	}

	ledgerMembers, err := loadExportSection("ledger_members", `
		SELECT user_id, role, created_at, updated_at
		FROM ledger_members
		WHERE ledger_id = ?
		ORDER BY created_at ASC, user_id ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	transactionDefaults, err := loadExportSection("transaction_defaults", `
		SELECT user_id, type, category_id, account_id, payer_user_id, visibility,
		       split_method, tag_names, updated_at
		FROM transaction_defaults
		WHERE ledger_id = ? AND user_id = ?
		ORDER BY user_id ASC
	`, ledgerID, actorUserID)
	if err != nil {
		return nil, err
	}
	transactionTags, err := loadExportSection("transaction_tags", `
		SELECT links.transaction_id, links.tag_id
		FROM transaction_tags links
		JOIN transactions parent ON parent.id = links.transaction_id
		JOIN tags tag ON tag.id = links.tag_id
		WHERE parent.ledger_id = ?
		  AND tag.ledger_id = ?
		  AND parent.status != 'deleted'
		  AND (
			parent.created_by_user_id = ?
			OR parent.owner_user_id = ?
			OR parent.payer_user_id = ?
			OR parent.visibility IN ('partner_readable', 'shared')
		  )
		ORDER BY links.transaction_id ASC, links.tag_id ASC
	`, ledgerID, ledgerID, actorUserID, actorUserID, actorUserID)
	if err != nil {
		return nil, err
	}
	transactionTemplates, err := loadExportSection("transaction_templates", `
		SELECT id, ledger_id, name, type, title, amount_cents, category_id, account_id,
		       payer_user_id, split_method, tag_names, note, created_by_user_id,
		       is_archived, archived_at, created_at, updated_at
		FROM transaction_templates
		WHERE ledger_id = ?
		ORDER BY created_at ASC, id ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	recurringRules, err := loadExportSection("recurring_rules", `
		SELECT id, ledger_id, name, type, title, amount_cents, category_id,
		       payer_user_id, split_method, tag_names, note, frequency,
		       next_due_date, created_by_user_id, created_at, updated_at
		FROM recurring_rules
		WHERE ledger_id = ?
		ORDER BY created_at ASC, id ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	recurringReminders, err := loadExportSection("recurring_reminders", `
		SELECT reminder.id, reminder.ledger_id, reminder.rule_id, reminder.due_date,
		       reminder.status, reminder.transaction_id, reminder.created_at,
		       reminder.updated_at
		FROM recurring_reminders reminder
		JOIN recurring_rules rule
		  ON rule.id = reminder.rule_id
		 AND rule.ledger_id = reminder.ledger_id
		WHERE reminder.ledger_id = ?
		ORDER BY reminder.due_date ASC, reminder.id ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	importBatches, err := loadExportSection("import_batches", `
		SELECT id, ledger_id, filename, created_by_user_id, status, source_type,
		       file_sha256, total_rows, new_rows, duplicate_rows, suspicious_rows,
		       invalid_rows, imported_rows, skipped_rows, failed_rows, file_format,
		       parser_metadata_json, created_at, updated_at, committed_at, expires_at
		FROM import_batches
		WHERE ledger_id = ?
		ORDER BY created_at ASC, id ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	importItems, err := loadExportSection("import_items", `
		SELECT item.id, item.batch_id, item.transaction_id, item.import_hash,
		       item.status, item.row_number, item.source_type, item.external_order_id,
		       item.occurred_at, item.title, item.merchant, item.description,
		       item.amount_cents, item.direction, item.target_transaction_type,
		       item.duplicate_status, item.row_status, item.normalized_json,
		       item.user_adjustment_json, item.error_code, item.error_message,
		       item.generated_transaction_id, item.suggested_category_id,
		       item.suggested_account_id, item.suggested_tag_ids_json,
		       item.selected_category_id, item.selected_account_id,
		       item.selected_tag_ids_json, item.visibility, item.suggested_rule_id,
		       item.suggestion_reason, item.created_at
		FROM import_items item
		JOIN import_batches batch ON batch.id = item.batch_id
		WHERE batch.ledger_id = ?
		ORDER BY batch.created_at ASC, item.row_number ASC, item.id ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}
	transactionImportRefs, err := loadExportSection("transaction_import_refs", `
		SELECT ref.id, ref.ledger_id, ref.transaction_id, ref.import_batch_id,
		       ref.import_row_id, ref.import_hash, ref.external_order_id,
		       ref.source_type, ref.created_at
		FROM transaction_import_refs ref
		JOIN transactions parent
		  ON parent.id = ref.transaction_id
		 AND parent.ledger_id = ref.ledger_id
		JOIN import_batches batch
		  ON batch.id = ref.import_batch_id
		 AND batch.ledger_id = ref.ledger_id
		WHERE ref.ledger_id = ?
		  AND parent.status != 'deleted'
		  AND (
			parent.created_by_user_id = ?
			OR parent.owner_user_id = ?
			OR parent.payer_user_id = ?
			OR parent.visibility IN ('partner_readable', 'shared')
		  )
		ORDER BY ref.created_at ASC, ref.id ASC
	`, ledgerID, actorUserID, actorUserID, actorUserID)
	if err != nil {
		return nil, err
	}
	importRules, err := loadExportSection("import_rules", `
		SELECT id, ledger_id, keyword, category_id, tag_names, account_id,
		       created_by_user_id, name, match_type, pattern, amount_min_cents,
		       amount_max_cents, priority, result_json, status, archived_at,
		       created_at, updated_at
		FROM import_rules
		WHERE ledger_id = ?
		ORDER BY priority ASC, created_at ASC, id ASC
	`, ledgerID)
	if err != nil {
		return nil, err
	}

	var ledgerName, defaultCurrency, ledgerStatus, ledgerCreatedAt, ledgerUpdatedAt string
	var ledgerVersion int64
	var archivedAt, archivedByUserID sql.NullString
	if err := s.db.QueryRowContext(ctx, `
		SELECT name, default_currency, status, version, archived_at,
		       archived_by_user_id, created_at, updated_at
		FROM ledgers
		WHERE id = ?
	`, ledgerID).Scan(
		&ledgerName,
		&defaultCurrency,
		&ledgerStatus,
		&ledgerVersion,
		&archivedAt,
		&archivedByUserID,
		&ledgerCreatedAt,
		&ledgerUpdatedAt,
	); err != nil {
		return nil, errors.NewAppError(
			http.StatusInternalServerError,
			errors.ErrCodeExportFailed,
			"query export ledger manifest: "+err.Error(),
		)
	}
	schemaVersion, err := goose.GetDBVersion(s.db)
	if err != nil {
		return nil, errors.NewAppError(
			http.StatusInternalServerError,
			errors.ErrCodeExportFailed,
			"query export schema version: "+err.Error(),
		)
	}
	ledgerManifest := map[string]any{
		"id":               ledgerID,
		"name":             ledgerName,
		"default_currency": defaultCurrency,
		"status":           ledgerStatus,
		"version":          ledgerVersion,
		"created_at":       ledgerCreatedAt,
		"updated_at":       ledgerUpdatedAt,
	}
	if archivedAt.Valid {
		ledgerManifest["archived_at"] = archivedAt.String
	}
	if archivedByUserID.Valid {
		ledgerManifest["archived_by_user_id"] = archivedByUserID.String
	}
	manifest := map[string]any{
		"format":         "ledger_two_ledger_export",
		"format_version": 1,
		"purpose":        "portable_read_only_snapshot",
		"restorable":     false,
		"exported_at":    time.Now().UTC().Format(time.RFC3339),
		"schema_version": schemaVersion,
		"ledger":         ledgerManifest,
		"actor": map[string]any{
			"user_id": actorUserID,
			"role":    string(lc.Role),
		},
	}

	// 组装最终结果
	exportData := map[string]interface{}{
		"manifest":                manifest,
		"ledger_members":          ledgerMembers,
		"users":                   users,
		"categories":              categories,
		"tags":                    tags,
		"accounts":                accounts,
		"transaction_defaults":    transactionDefaults,
		"transactions":            transactions,
		"transaction_tags":        transactionTags,
		"transaction_splits":      splits,
		"settlements":             settlements,
		"transaction_templates":   transactionTemplates,
		"recurring_rules":         recurringRules,
		"recurring_reminders":     recurringReminders,
		"import_batches":          importBatches,
		"import_items":            importItems,
		"transaction_import_refs": transactionImportRefs,
		"import_rules":            importRules,
		"audit_logs":              auditLogs,
	}

	jsonBytes, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeExportFailed, "failed to marshal JSON export: "+err.Error())
	}

	// 写入审计日志
	afterObj := map[string]interface{}{
		"export_type":      "json",
		"format_version":   1,
		"user_records":     len(users),
		"tx_records":       len(transactions),
		"import_records":   len(importItems),
		"template_records": len(transactionTemplates),
	}
	afterBytes, _ := json.Marshal(afterObj)

	auditQuery := `
		INSERT INTO audit_logs (id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at)
		VALUES (?, ?, ?, 'export_json', 'database', 'json', NULL, ?, ?)
	`
	_, _ = s.db.ExecContext(ctx, auditQuery,
		uuid.NewString(), ledgerID, actorUserID, string(afterBytes), time.Now().Format(time.RFC3339))

	return jsonBytes, nil
}
