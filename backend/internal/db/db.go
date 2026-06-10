package db

import (
	"database/sql"
	"fmt"
	"log"

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

	log.Printf("Running database migrations...")
	if err := goose.Up(database, "."); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Printf("Database migrations completed successfully")

	return database, nil
}
