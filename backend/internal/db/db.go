package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/migrations"
)

// Init 负责初始化 SQLite 数据库连接并自动运行数据库迁移
func Init(dsn string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 自动运行数据表迁移，确保表结构为最新版
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	currentVersion, err := goose.GetDBVersion(database)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to read database schema version: %w", err)
	}

	task50Snapshot, err := prepareTask50Upgrade(context.Background(), database, currentVersion)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("database migration preflight failed: %w", err)
	}

	if currentVersion > 0 {
		// 在执行可能有风险的 Migration 之前进行防破坏自动备份
		// 提取物理路径 (粗略移除可能存在的 URI 协议与参数)
		dbPath := dsn
		if idx := strings.Index(dbPath, "?"); idx != -1 {
			dbPath = dbPath[:idx]
		}
		dbPath = strings.TrimPrefix(dbPath, "file:")

		bakPath := fmt.Sprintf("%s.pre_migrate_v%d.bak", dbPath, currentVersion)
		if _, err := os.Stat(bakPath); os.IsNotExist(err) {
			if src, err := os.Open(dbPath); err == nil {
				if dst, err := os.Create(bakPath); err == nil {
					io.Copy(dst, src)
					dst.Close()
					log.Printf("Safety Check: Auto-backed up database before migration to %s", bakPath)
				}
				src.Close()
			}
		}
	}

	log.Printf("Running database migrations...")
	if err := goose.Up(database, "."); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	if err := verifyTask50MigrationSnapshot(context.Background(), database, task50Snapshot); err != nil {
		database.Close()
		return nil, fmt.Errorf("database migration conservation check failed: %w", err)
	}
	log.Printf("Database migrations completed successfully")

	return database, nil
}
