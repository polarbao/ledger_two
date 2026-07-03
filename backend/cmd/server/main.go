package main

import (
	"log"
	"net/http"

	"ledger_two/internal/config"
	"ledger_two/internal/db"
	"ledger_two/internal/http/router"
)

func main() {
	cfg := config.Load()
	if err := cfg.ValidateRuntime(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// 正式接通真实的 SQLite
	database, err := db.Init(cfg.DSN)
	if err != nil {
		log.Fatalf("failed to initialize db: %v", err)
	}
	defer database.Close()

	// 将 DB 连接和配置实例全部挂载进入核心容器中
	r := router.New(database, cfg)

	log.Printf("Server starting on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
