package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("DEPLOYMENT_CHANNEL", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("PORT", "")
	t.Setenv("DB_DSN", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("BACKUP_DIR", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("LOG_DIR", "")
	unsetEnv(t, "COOKIE_SECURE")

	cfg := Load()
	if cfg.Env != "development" {
		t.Fatalf("expected development env, got %s", cfg.Env)
	}
	if cfg.DeploymentChannel != DeploymentChannelDevelopment {
		t.Fatalf("expected development deployment channel, got %s", cfg.DeploymentChannel)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default HTTPAddr :8080, got %s", cfg.HTTPAddr)
	}
	if cfg.JWTSecret != DevJWTSecret {
		t.Fatalf("expected dev jwt secret fallback")
	}
	if err := cfg.ValidateRuntime(); err != nil {
		t.Fatalf("development validation should pass: %v", err)
	}
}

func TestValidateRuntimeRejectsWeakProductionSecret(t *testing.T) {
	tmp := t.TempDir()
	weakSecrets := []string{
		DevJWTSecret,
		"please-change-me-to-a-64-character-random-secret-before-production",
	}
	for _, secret := range weakSecrets {
		t.Run(secret, func(t *testing.T) {
			t.Setenv("APP_ENV", "production")
			t.Setenv("HTTP_ADDR", ":8080")
			t.Setenv("DB_DSN", filepath.Join(tmp, "data", "ledger.db"))
			t.Setenv("BACKUP_DIR", filepath.Join(tmp, "backups"))
			t.Setenv("UPLOAD_DIR", filepath.Join(tmp, "uploads"))
			t.Setenv("LOG_DIR", filepath.Join(tmp, "logs"))
			t.Setenv("JWT_SECRET", secret)
			t.Setenv("COOKIE_SECURE", "false")

			cfg := Load()
			err := cfg.ValidateRuntime()
			if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
				t.Fatalf("expected weak JWT_SECRET validation error, got %v", err)
			}
		})
	}
}

func TestValidateRuntimeRequiresExplicitCookieSecure(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_DSN", filepath.Join(tmp, "data", "ledger.db"))
	t.Setenv("BACKUP_DIR", filepath.Join(tmp, "backups"))
	t.Setenv("UPLOAD_DIR", filepath.Join(tmp, "uploads"))
	t.Setenv("LOG_DIR", filepath.Join(tmp, "logs"))
	t.Setenv("JWT_SECRET", "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz")
	unsetEnv(t, "COOKIE_SECURE")

	cfg := Load()
	err := cfg.ValidateRuntime()
	if err == nil || !strings.Contains(err.Error(), "COOKIE_SECURE") {
		t.Fatalf("expected COOKIE_SECURE validation error, got %v", err)
	}
}

func TestValidateRuntimeAcceptsProductionConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APP_ENV", "production")
	t.Setenv("DEPLOYMENT_CHANNEL", "staging")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("APP_BASE_URL", "http://nas.local:38088")
	t.Setenv("DB_DSN", filepath.Join(tmp, "data", "ledger.db"))
	t.Setenv("BACKUP_DIR", filepath.Join(tmp, "backups"))
	t.Setenv("UPLOAD_DIR", filepath.Join(tmp, "uploads"))
	t.Setenv("LOG_DIR", filepath.Join(tmp, "logs"))
	t.Setenv("JWT_SECRET", "0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_SAMESITE", "Lax")

	cfg := Load()
	if err := cfg.ValidateRuntime(); err != nil {
		t.Fatalf("production validation should pass: %v", err)
	}
	if cfg.DeploymentChannel != DeploymentChannelStaging {
		t.Fatalf("expected staging deployment channel, got %s", cfg.DeploymentChannel)
	}
}

func TestLoadDefaultsProductionDeploymentChannel(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DEPLOYMENT_CHANNEL", "")
	unsetEnv(t, "IMPORT_XLSX_ENABLED")

	cfg := Load()
	if cfg.DeploymentChannel != DeploymentChannelProduction {
		t.Fatalf("expected production deployment channel, got %s", cfg.DeploymentChannel)
	}
	if cfg.ImportXLSXEnabled {
		t.Fatalf("expected XLSX imports disabled by default in production")
	}
}

func TestLoadXLSXImportGateByDeploymentChannel(t *testing.T) {
	testCases := []struct {
		name    string
		channel string
		enabled bool
	}{
		{name: "development", channel: DeploymentChannelDevelopment, enabled: true},
		{name: "staging", channel: DeploymentChannelStaging, enabled: true},
		{name: "production", channel: DeploymentChannelProduction, enabled: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("APP_ENV", "production")
			t.Setenv("DEPLOYMENT_CHANNEL", testCase.channel)
			unsetEnv(t, "IMPORT_XLSX_ENABLED")

			cfg := Load()
			if cfg.ImportXLSXEnabled != testCase.enabled {
				t.Fatalf("expected IMPORT_XLSX_ENABLED=%v for %s, got %v", testCase.enabled, testCase.channel, cfg.ImportXLSXEnabled)
			}
		})
	}
}

func TestLoadXLSXImportGateAllowsExplicitOverride(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DEPLOYMENT_CHANNEL", DeploymentChannelProduction)
	t.Setenv("IMPORT_XLSX_ENABLED", "true")

	cfg := Load()
	if !cfg.ImportXLSXEnabled {
		t.Fatalf("expected explicit production XLSX override to be enabled")
	}
}

func TestValidateRuntimeRejectsInvalidXLSXImportGate(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DEPLOYMENT_CHANNEL", DeploymentChannelDevelopment)
	t.Setenv("IMPORT_XLSX_ENABLED", "sometimes")

	cfg := Load()
	err := cfg.ValidateRuntime()
	if err == nil || !strings.Contains(err.Error(), "IMPORT_XLSX_ENABLED") {
		t.Fatalf("expected IMPORT_XLSX_ENABLED validation error, got %v", err)
	}
}

func TestValidateRuntimeRejectsUnknownDeploymentChannel(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DEPLOYMENT_CHANNEL", "preview")

	cfg := Load()
	err := cfg.ValidateRuntime()
	if err == nil || !strings.Contains(err.Error(), "DEPLOYMENT_CHANNEL") {
		t.Fatalf("expected deployment channel validation error, got %v", err)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	previous, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, previous)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
