package config

import (
	"os"
)

type Config struct {
	Port      string
	DSN       string
	JWTSecret string
	BackupDir string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "data/ledger.db"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "ledger-two-secret-for-dev-only"
	}
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "data/backups"
	}
	return &Config{
		Port:      port,
		DSN:       dsn,
		JWTSecret: jwtSecret,
		BackupDir: backupDir,
	}
}
