package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DevJWTSecret = "ledger-two-secret-for-dev-only"

const (
	DeploymentChannelDevelopment = "development"
	DeploymentChannelStaging     = "staging"
	DeploymentChannelProduction  = "production"
)

type Config struct {
	Env               string
	DeploymentChannel string
	Port              string
	HTTPAddr          string
	AppBaseURL        string
	DSN               string
	JWTSecret         string
	BackupDir         string
	UploadDir         string
	LogDir            string
	CookieSecure      string
	CookieSameSite    string
	cookieSecureSet   bool
}

func Load() *Config {
	env := getenvDefault("APP_ENV", "development")
	deploymentChannel := os.Getenv("DEPLOYMENT_CHANNEL")
	if deploymentChannel == "" {
		if env == "development" {
			deploymentChannel = DeploymentChannelDevelopment
		} else {
			deploymentChannel = DeploymentChannelProduction
		}
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":" + port
	}

	dsn := getenvDefault("DB_DSN", "data/ledger.db")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = DevJWTSecret
	}
	cookieSecure, cookieSecureSet := os.LookupEnv("COOKIE_SECURE")

	return &Config{
		Env:               env,
		DeploymentChannel: deploymentChannel,
		Port:              port,
		HTTPAddr:          httpAddr,
		AppBaseURL:        os.Getenv("APP_BASE_URL"),
		DSN:               dsn,
		JWTSecret:         jwtSecret,
		BackupDir:         getenvDefault("BACKUP_DIR", "data/backups"),
		UploadDir:         getenvDefault("UPLOAD_DIR", "data/uploads"),
		LogDir:            getenvDefault("LOG_DIR", "data/logs"),
		CookieSecure:      cookieSecure,
		CookieSameSite:    getenvDefault("COOKIE_SAMESITE", "Lax"),
		cookieSecureSet:   cookieSecureSet,
	}
}

func (c *Config) ValidateRuntime() error {
	if !isValidDeploymentChannel(c.DeploymentChannel) {
		return fmt.Errorf("DEPLOYMENT_CHANNEL must be development, staging, or production")
	}
	if c.Env != "production" {
		return nil
	}

	if isWeakProductionSecret(c.JWTSecret) {
		return fmt.Errorf("JWT_SECRET must be a strong random value in production")
	}
	if !c.cookieSecureSet {
		return fmt.Errorf("COOKIE_SECURE must be explicitly set in production")
	}
	if c.CookieSecure != "true" && c.CookieSecure != "false" {
		return fmt.Errorf("COOKIE_SECURE must be true or false")
	}
	if err := validateSQLiteParentDir(c.DSN); err != nil {
		return fmt.Errorf("DB_DSN parent directory is not writable: %w", err)
	}
	for name, dir := range map[string]string{
		"BACKUP_DIR": c.BackupDir,
		"UPLOAD_DIR": c.UploadDir,
		"LOG_DIR":    c.LogDir,
	} {
		if err := ensureWritableDir(dir); err != nil {
			return fmt.Errorf("%s is not writable: %w", name, err)
		}
	}
	return nil
}

func isValidDeploymentChannel(channel string) bool {
	switch channel {
	case DeploymentChannelDevelopment, DeploymentChannelStaging, DeploymentChannelProduction:
		return true
	default:
		return false
	}
}

func getenvDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func isWeakProductionSecret(secret string) bool {
	trimmed := strings.TrimSpace(secret)
	if len(trimmed) < 32 {
		return true
	}
	weakValues := map[string]bool{
		DevJWTSecret: true,
		"change-this-to-a-long-random-secret-key":                            true,
		"replace-with-a-long-random-string":                                  true,
		"please-change-me":                                                   true,
		"please-change-me-to-a-64-character-random-secret-before-production": true,
		"please-change-me-to-a-long-random-secret":                           true,
		"please-change-me-to-a-long-random-secret-key":                       true,
	}
	return weakValues[trimmed]
}

func validateSQLiteParentDir(dsn string) error {
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file::memory:") {
		return nil
	}
	path := dsn
	if strings.HasPrefix(path, "file:") {
		path = strings.TrimPrefix(path, "file:")
		if idx := strings.Index(path, "?"); idx >= 0 {
			path = path[:idx]
		}
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return ensureWritableDir(dir)
}

func ensureWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	testFile, err := os.CreateTemp(dir, ".write_test_*")
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
