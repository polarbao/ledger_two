package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Init 负责初始化 SQLite 数据库连接。
// 现阶段仅构建占位连接函数，待执行 Database Migration（Task 02）时再补充配置。
func Init(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite3", dsn)
}
