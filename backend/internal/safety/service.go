package safety

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

	if backups, err := s.GetBackups(ctx); err == nil && len(backups) > 0 {
		diag.LatestBackup = &backups[0]
	}
	diag.AuditActionCount = s.auditActionCounts(ctx)
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
func (s *Service) ManualBackup(ctx context.Context, actorUserID string) (string, error) {
	if err := s.requireInstanceAdmin(ctx, actorUserID); err != nil {
		return "", err
	}

	backupDir := s.cfg.BackupDir
	manualDir := filepath.Join(backupDir, "manual")
	if err := s.checkBackupDirWritable(manualDir); err != nil {
		return "", err
	}

	now := time.Now()
	// 避免文件名中包含空格或冒号，采用简洁的时间格式
	filename := fmt.Sprintf("backup_manual_%s_%d.db", now.Format("20060102_150405"), now.Nanosecond())
	backupPath := filepath.Join(manualDir, filename)

	// 使用 VACUUM INTO 进行安全在线备份，防止热拷贝锁死
	safePath := strings.ReplaceAll(backupPath, "'", "''")
	query := fmt.Sprintf("VACUUM INTO '%s'", safePath)
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "database vacuum into failed: "+err.Error())
	}

	// 获取文件大小
	var sizeBytes int64
	if fi, err := os.Stat(backupPath); err == nil {
		sizeBytes = fi.Size()
	}

	// 记录实例级审计日志，不伪造账本 ID。
	afterJSONObj := map[string]interface{}{
		"filename":   filename,
		"size_bytes": sizeBytes,
		"rel_path":   "manual/" + filename,
	}
	afterBytes, _ := json.Marshal(afterJSONObj)

	if err := s.recordInstanceAudit(ctx, actorUserID, "manual_database_backup", "database", filename, string(afterBytes), now); err != nil {
		_ = os.Remove(backupPath)
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "failed to record backup audit log: "+err.Error())
	}

	return "manual/" + filename, nil
}

// RestoreBackup 准备恢复流程。因操作系统文件锁机制，后端仅执行自动前置备份并返回操作指引。
func (s *Service) RestoreBackup(ctx context.Context, actorUserID string, targetFilename string) (string, error) {
	if err := s.requireInstanceAdmin(ctx, actorUserID); err != nil {
		return "", err
	}

	backupDir := s.cfg.BackupDir
	targetPath := filepath.Join(backupDir, filepath.Clean(targetFilename))

	// 防御：路径穿越或文件不存在
	if !strings.HasPrefix(targetPath, filepath.Clean(backupDir)) {
		return "", errors.NewAppError(http.StatusForbidden, "FORBIDDEN", "无权访问该路径下的物理文件")
	}
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return "", errors.NewAppError(http.StatusNotFound, "NOT_FOUND", "目标备份文件不存在")
	}

	// 1. 强制执行前置备份 (后悔药)
	now := time.Now()
	manualDir := filepath.Join(backupDir, "manual")
	if err := s.checkBackupDirWritable(manualDir); err != nil {
		return "", err
	}

	preFilename := fmt.Sprintf("pre_restore_%s_%d.db", now.Format("20060102_150405"), now.Nanosecond())
	preBackupPath := filepath.Join(manualDir, preFilename)

	safePath := strings.ReplaceAll(preBackupPath, "'", "''")
	query := fmt.Sprintf("VACUUM INTO '%s'", safePath)
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "自动前置备份失败，终止恢复流程: "+err.Error())
	}

	afterJSONObj := map[string]interface{}{
		"action":          "pre_restore_backup",
		"target_restore":  targetFilename,
		"pre_backup_file": "manual/" + preFilename,
	}
	afterBytes, _ := json.Marshal(afterJSONObj)
	if err := s.recordInstanceAudit(ctx, actorUserID, "prepare_database_restore", "database", targetFilename, string(afterBytes), now); err != nil {
		_ = os.Remove(preBackupPath)
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "记录恢复准备审计失败")
	}

	// 2. 返回人工覆盖指引（策略 B）
	instructions := fmt.Sprintf("已自动为您创建当前数据的安全备份 (%s)。为避免数据损坏，请关闭程序后，手动将文件 %s 覆盖至 data/ledger.db，随后重新启动服务即可生效。", preFilename, targetFilename)

	return instructions, nil
}

// GetBackups 获取所有备份文件列表
func (s *Service) GetBackups(ctx context.Context) ([]BackupInfo, error) {
	backupDir := s.cfg.BackupDir
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		_ = os.MkdirAll(backupDir, 0755)
	}

	list := make([]BackupInfo, 0)
	err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".db") {
			return nil
		}

		rel, err := filepath.Rel(backupDir, path)
		if err != nil {
			rel = info.Name()
		}
		rel = filepath.ToSlash(rel)

		list = append(list, BackupInfo{
			Filename:  rel,
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "failed to scan backups: "+err.Error())
	}

	// 降序排序（最新修改的备份在前）
	sort.Slice(list, func(i, j int) bool {
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})

	return list, nil
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
		SELECT users.id, users.display_name, users.username
		FROM ledger_members
		JOIN users ON users.id = ledger_members.user_id
		WHERE ledger_members.ledger_id = ?
	`, ledgerID)
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
		SELECT users.id, users.username, users.display_name, users.avatar_url,
		       ledger_members.role, users.is_active, users.created_at, users.updated_at
		FROM ledger_members
		JOIN users ON users.id = ledger_members.user_id
		WHERE ledger_members.ledger_id = ?
		ORDER BY ledger_members.created_at ASC, users.id ASC
	`, ledgerID)
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
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}
	categories := make([]CategoryDTO, 0)
	cRows, err := s.db.QueryContext(ctx, "SELECT id, ledger_id, owner_user_id, name, type, icon, color, parent_id, sort_order, is_system, created_at, updated_at FROM categories WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var c CategoryDTO
			var owner, icon, color, parent sql.NullString
			err := cRows.Scan(&c.ID, &c.LedgerID, &owner, &c.Name, &c.Type, &icon, &color, &parent, &c.SortOrder, &c.IsSystem, &c.CreatedAt, &c.UpdatedAt)
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
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}
	tags := make([]TagDTO, 0)
	tRows, err := s.db.QueryContext(ctx, "SELECT id, ledger_id, name, owner_user_id, color, created_at, updated_at FROM tags WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var t TagDTO
			var owner, color sql.NullString
			if err := tRows.Scan(&t.ID, &t.LedgerID, &t.Name, &owner, &color, &t.CreatedAt, &t.UpdatedAt); err == nil {
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
		IsArchived     int    `json:"is_archived"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	}
	accounts := make([]AccountDTO, 0)
	aRows, err := s.db.QueryContext(ctx, "SELECT id, ledger_id, owner_user_id, name, type, currency, initial_balance, is_archived, created_at, updated_at FROM accounts WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a AccountDTO
			if err := aRows.Scan(&a.ID, &a.LedgerID, &a.OwnerUserID, &a.Name, &a.Type, &a.Currency, &a.InitialBalance, &a.IsArchived, &a.CreatedAt, &a.UpdatedAt); err == nil {
				accounts = append(accounts, a)
			}
		}
	}

	// 5. 查询当前用户可见的 transactions (排除 status = 'deleted')
	type TransactionDTO struct {
		ID              string  `json:"id"`
		LedgerID        string  `json:"ledger_id"`
		Type            string  `json:"type"`
		Title           string  `json:"title"`
		Amount          int64   `json:"amount"`
		Currency        string  `json:"currency"`
		OccurredAt      string  `json:"occurred_at"`
		OwnerUserID     string  `json:"owner_user_id"`
		CreatedByUserID string  `json:"created_by_user_id"`
		PayerUserID     *string `json:"payer_user_id,omitempty"`
		AccountID       *string `json:"account_id,omitempty"`
		CategoryID      *string `json:"category_id,omitempty"`
		Visibility      string  `json:"visibility"`
		SplitMethod     *string `json:"split_method,omitempty"`
		Note            *string `json:"note,omitempty"`
		Status          string  `json:"status"`
		CreatedAt       string  `json:"created_at"`
		UpdatedAt       string  `json:"updated_at"`
	}
	transactions := make([]TransactionDTO, 0)
	visibleTxIDs := make(map[string]bool)

	txQuery := `
		SELECT 
			id, ledger_id, type, title, amount, currency, occurred_at, 
			owner_user_id, created_by_user_id, payer_user_id, account_id, category_id, 
			visibility, split_method, note, status, created_at, updated_at 
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
			var payer, account, category, split, note sql.NullString
			err := txRows.Scan(&tx.ID, &tx.LedgerID, &tx.Type, &tx.Title, &tx.Amount, &tx.Currency, &tx.OccurredAt,
				&tx.OwnerUserID, &tx.CreatedByUserID, &payer, &account, &category,
				&tx.Visibility, &split, &note, &tx.Status, &tx.CreatedAt, &tx.UpdatedAt)
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
		Action      string  `json:"action"`
		EntityType  string  `json:"entity_type"`
		EntityID    string  `json:"entity_id"`
		BeforeJSON  *string `json:"before_json,omitempty"`
		AfterJSON   *string `json:"after_json,omitempty"`
		CreatedAt   string  `json:"created_at"`
	}
	auditLogs := make([]AuditLogDTO, 0)
	aRows, err = s.db.QueryContext(ctx, "SELECT id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at FROM audit_logs WHERE ledger_id = ?", ledgerID)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var a AuditLogDTO
			var before, after sql.NullString
			err := aRows.Scan(&a.ID, &a.LedgerID, &a.ActorUserID, &a.Action, &a.EntityType, &a.EntityID, &before, &after, &a.CreatedAt)
			if err == nil {
				// 权限收拢过滤：若为交易实体审计，且该交易对当前登录人不可见，则安全排除该条审计行
				if a.EntityType == "transaction" && !visibleTxIDs[a.EntityID] {
					continue
				}
				if before.Valid {
					a.BeforeJSON = &before.String
				}
				if after.Valid {
					a.AfterJSON = &after.String
				}
				auditLogs = append(auditLogs, a)
			}
		}
	}

	// 组装最终结果
	exportData := map[string]interface{}{
		"users":              users,
		"categories":         categories,
		"tags":               tags,
		"accounts":           accounts,
		"transactions":       transactions,
		"transaction_splits": splits,
		"settlements":        settlements,
		"audit_logs":         auditLogs,
	}

	jsonBytes, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeExportFailed, "failed to marshal JSON export: "+err.Error())
	}

	// 写入审计日志
	afterObj := map[string]interface{}{
		"export_type":  "json",
		"user_records": len(users),
		"tx_records":   len(transactions),
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
