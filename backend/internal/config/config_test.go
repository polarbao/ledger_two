package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	// 创建临时目录测试
	tempDir, err := os.MkdirTemp("", "ledger_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validDSN := filepath.Join(tempDir, "ledger.db")
	validBackup := filepath.Join(tempDir, "backups")
	validUpload := filepath.Join(tempDir, "uploads")

	t.Run("Development mode bypasses strict check", func(t *testing.T) {
		cfg := &Config{
			AppEnv:    "development",
			JWTSecret: "ledger-two-secret-for-dev-only", // 开发默认密钥
			DSN:       validDSN,
			BackupDir: validBackup,
			UploadDir: validUpload,
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("expected no error in development mode, got: %v", err)
		}
	})

	t.Run("Production mode rejects default dev secret", func(t *testing.T) {
		cfg := &Config{
			AppEnv:    "production",
			JWTSecret: "ledger-two-secret-for-dev-only",
			DSN:       validDSN,
			BackupDir: validBackup,
			UploadDir: validUpload,
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for default dev secret in production, got nil")
		}
	})

	t.Run("Production mode rejects short secret", func(t *testing.T) {
		cfg := &Config{
			AppEnv:    "production",
			JWTSecret: "short-secret-less-than-32",
			DSN:       validDSN,
			BackupDir: validBackup,
			UploadDir: validUpload,
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for short secret in production, got nil")
		}
	})

	t.Run("Production mode accepts strong secret and valid dirs", func(t *testing.T) {
		// 生成 32 字节强密钥
		strongSecret := "12345678901234567890123456789012"
		cfg := &Config{
			AppEnv:    "production",
			JWTSecret: strongSecret,
			DSN:       validDSN,
			BackupDir: validBackup,
			UploadDir: validUpload,
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("expected validation success for strong secret, got error: %v", err)
		}
	})

	t.Run("Production mode rejects invalid backup dir", func(t *testing.T) {
		strongSecret := "12345678901234567890123456789012"
		// 使用一个根本不存在或者无权创建的根路径目录（比如只读权限或者非法路径）
		invalidDir := "/sys/kernel/security/ledger_test_invalid"
		cfg := &Config{
			AppEnv:    "production",
			JWTSecret: strongSecret,
			DSN:       validDSN,
			BackupDir: invalidDir,
			UploadDir: validUpload,
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for read-only backup dir, got nil")
		}
	})
}
