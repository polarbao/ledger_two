package main

import (
	"flag"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	"ledger_two/internal/config"
	"ledger_two/internal/db"
	"ledger_two/migrations"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		args = []string{"up"}
	}

	command := args[0]

	cfg := config.Load()

	database, err := db.Init(cfg.DSN)
	if err != nil {
		log.Fatalf("failed to initialize db: %v", err)
	}
	defer database.Close()

	// 告知 Goose 优先使用嵌入的文件系统查找 sql 脚本
	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("failed to set goose dialect: %v", err)
	}

	// 传入 "." 表示使用 embed 的根目录
	if err := goose.Run(command, database, ".", args[1:]...); err != nil {
		log.Fatalf("goose %v: %v", command, err)
	}
}
