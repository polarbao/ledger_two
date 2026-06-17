package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppEnv         string
	Port           string // 对应 HTTP_ADDR
	BaseURL        string // 对应 APP_BASE_URL
	DSN            string // 对应 DB_DSN
	BackupDir      string // 对应 BACKUP_DIR
	UploadDir      string // 对应 UPLOAD_DIR
	LogDir         string // 对应 LOG_DIR
	JWTSecret      string // 对应 JWT_SECRET
	CookieSecure   string // 对应 COOKIE_SECURE
	CookieSameSite string // 对应 COOKIE_SAMESITE
}

func Load() *Config {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	port := os.Getenv("HTTP_ADDR")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = ":8080"
	}
	if !strings.HasPrefix(port, ":") && !strings.Contains(port, ".") {
		port = ":" + port
	}

	baseURL := os.Getenv("APP_BASE_URL")

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "data/ledger.db"
	}

	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "data/backups"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "data/logs"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = os.Getenv("SESSION_SECRET")
	}
	if jwtSecret == "" {
		jwtSecret = "ledger-two-secret-for-dev-only"
	}

	cookieSecure := os.Getenv("COOKIE_SECURE")
	cookieSameSite := os.Getenv("COOKIE_SAMESITE")
	if cookieSameSite == "" {
		cookieSameSite = "Lax"
	}

	return &Config{
		AppEnv:         appEnv,
		Port:           port,
		BaseURL:        baseURL,
		DSN:            dsn,
		BackupDir:      backupDir,
		UploadDir:      uploadDir,
		LogDir:         logDir,
		JWTSecret:      jwtSecret,
		CookieSecure:   cookieSecure,
		CookieSameSite: cookieSameSite,
	}
}

func (c *Config) Validate() error {
	if c.AppEnv == "production" {
		// 1. 强安全密钥校验
		if c.JWTSecret == "" || c.JWTSecret == "ledger-two-secret-for-dev-only" || c.JWTSecret == "replace-with-a-long-random-string" {
			return fmt.Errorf("安全漏洞隐患：生产环境必须配置自定义的强安全密钥(JWT_SECRET)")
		}
		if len(c.JWTSecret) < 32 {
			return fmt.Errorf("安全弱密钥：生产环境 JWT_SECRET 长度必须不少于 32 个字符，建议使用 64 位强随机字符串")
		}

		// 2. 目录可写校验
		dsnDir := filepath.Dir(c.DSN)
		if dsnDir != "." && dsnDir != "" {
			if err := os.MkdirAll(dsnDir, 0755); err != nil {
				return fmt.Errorf("创建 DSN 数据目录 %s 失败: %w", dsnDir, err)
			}
			tempFile := filepath.Join(dsnDir, ".write_test")
			if err := os.WriteFile(tempFile, []byte("ok"), 0644); err != nil {
				return fmt.Errorf("数据库目录 %s 无写入权限: %w", dsnDir, err)
			}
			_ = os.Remove(tempFile)
		}

		if err := os.MkdirAll(c.BackupDir, 0755); err != nil {
			return fmt.Errorf("创建备份目录 %s 失败: %w", c.BackupDir, err)
		}
		tempBackup := filepath.Join(c.BackupDir, ".write_test")
		if err := os.WriteFile(tempBackup, []byte("ok"), 0644); err != nil {
			return fmt.Errorf("备份目录 %s 无写入权限: %w", c.BackupDir, err)
		}
		_ = os.Remove(tempBackup)

		if err := os.MkdirAll(c.UploadDir, 0755); err != nil {
			return fmt.Errorf("创建上传目录 %s 失败: %w", c.UploadDir, err)
		}
		tempUpload := filepath.Join(c.UploadDir, ".write_test")
		if err := os.WriteFile(tempUpload, []byte("ok"), 0644); err != nil {
			return fmt.Errorf("上传目录 %s 无写入权限: %w", c.UploadDir, err)
		}
		_ = os.Remove(tempUpload)
	}
	return nil
}
