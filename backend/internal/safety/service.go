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
	"ledger_two/internal/http/middleware"
	ledgerctx "ledger_two/internal/ledger"

	"github.com/google/uuid"
)

type Service struct {
	db  *sql.DB
	cfg *config.Config
}

func NewService(db *sql.DB, cfg *config.Config) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

type BackupInfo struct {
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
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

// getUserLedgerID 获取唯一的账本 ID
func (s *Service) getUserLedgerID(ctx context.Context, userID string) (string, error) {
	if lc, ok := ledgerctx.LedgerContextFromContext(ctx); ok && lc.UserID == userID {
		return lc.LedgerID, nil
	}

	var id string

	headerLedgerID := middleware.GetHeaderLedgerIDFromContext(ctx)
	if headerLedgerID != "" {
		err := s.db.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE ledger_id = ? AND user_id = ?", headerLedgerID, userID).Scan(&id)
		if err != nil {
			return "", err
		}
		return id, nil
	}

	err := s.db.QueryRowContext(ctx, "SELECT ledger_id FROM ledger_members WHERE user_id = ? LIMIT 1", userID).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// ManualBackup 手动备份 SQLite 数据库
func (s *Service) ManualBackup(ctx context.Context, actorUserID string) (string, error) {
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

	ledgerID, err := s.getUserLedgerID(ctx, actorUserID)
	if err != nil {
		_ = os.Remove(backupPath)
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "failed to get default ledger: "+err.Error())
	}

	// 记录审计日志
	afterJSONObj := map[string]interface{}{
		"filename":   filename,
		"size_bytes": sizeBytes,
		"rel_path":   "manual/" + filename,
	}
	afterBytes, _ := json.Marshal(afterJSONObj)

	auditQuery := `
		INSERT INTO audit_logs (id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at)
		VALUES (?, ?, ?, 'backup', 'database', ?, NULL, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, auditQuery,
		uuid.NewString(), ledgerID, actorUserID, filename, string(afterBytes), now.Format(time.RFC3339))
	if err != nil {
		_ = os.Remove(backupPath)
		return "", errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeBackupFailed, "failed to record backup audit log: "+err.Error())
	}

	return "manual/" + filename, nil
}

// RestoreBackup 准备恢复流程。因操作系统文件锁机制，后端仅执行自动前置备份并返回操作指引。
func (s *Service) RestoreBackup(ctx context.Context, actorUserID string, targetFilename string) (string, error) {
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

	ledgerID, err := s.getUserLedgerID(ctx, actorUserID)
	if err == nil {
		afterJSONObj := map[string]interface{}{
			"action":          "pre_restore_backup",
			"target_restore":  targetFilename,
			"pre_backup_file": "manual/" + preFilename,
		}
		afterBytes, _ := json.Marshal(afterJSONObj)

		auditQuery := `
			INSERT INTO audit_logs (id, ledger_id, actor_user_id, action, entity_type, entity_id, before_json, after_json, created_at)
			VALUES (?, ?, ?, 'restore_prepare', 'database', ?, NULL, ?, ?)
		`
		_, _ = s.db.ExecContext(ctx, auditQuery,
			uuid.NewString(), ledgerID, actorUserID, targetFilename, string(afterBytes), now.Format(time.RFC3339))
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

	var list []BackupInfo
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
	ledgerID, err := s.getUserLedgerID(ctx, actorUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "failed to get default ledger: "+err.Error())
	}

	// 1. 查询用户
	userMap := make(map[string]string) // id -> display_name
	uRows, err := s.db.QueryContext(ctx, "SELECT id, display_name, username FROM users")
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
		WHERE t.ledger_id = ?
	`, ledgerID)
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
	ledgerID, err := s.getUserLedgerID(ctx, actorUserID)
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, errors.ErrCodeInternalError, "failed to get default ledger: "+err.Error())
	}

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
	uRows, err := s.db.QueryContext(ctx, "SELECT id, username, display_name, avatar_url, role, is_active, created_at, updated_at FROM users")
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
	sRows, err := s.db.QueryContext(ctx, "SELECT id, transaction_id, user_id, share_amount, share_ratio, created_at, updated_at FROM transaction_splits")
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

	// 9. 导出 app_settings
	type AppSettingDTO struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		UpdatedAt string `json:"updated_at"`
	}
	appSettings := make([]AppSettingDTO, 0)
	setSettingsRows, err := s.db.QueryContext(ctx, "SELECT key, value, updated_at FROM app_settings")
	if err == nil {
		defer setSettingsRows.Close()
		for setSettingsRows.Next() {
			var as AppSettingDTO
			if err := setSettingsRows.Scan(&as.Key, &as.Value, &as.UpdatedAt); err == nil {
				appSettings = append(appSettings, as)
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
		"app_settings":       appSettings,
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
