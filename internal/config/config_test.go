package config

import (
	"os"
	"testing"
)

// envKeys 是 Load 读取的全部环境变量。
var envKeys = []string{"F1GUIDE_HTTP_ADDR", "F1GUIDE_LOG_LEVEL", "F1GUIDE_DATABASE_URL"}

func unsetAll(t *testing.T) {
	t.Helper()
	for _, key := range envKeys {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
}

func TestLoadDefaults(t *testing.T) {
	unsetAll(t)

	cfg := Load()

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.DatabaseURL != "postgres://f1guide:f1guide@localhost:5432/f1guide?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want 本地开发默认连接串", cfg.DatabaseURL)
	}
}

func TestLoadFromEnv(t *testing.T) {
	unsetAll(t)
	t.Setenv("F1GUIDE_HTTP_ADDR", ":9090")
	t.Setenv("F1GUIDE_LOG_LEVEL", "debug")

	cfg := Load()

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9090")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	// 未设置的环境变量回退默认值
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL 为空，应回退默认值")
	}
}

func TestEmptyEnvFallsBackToDefault(t *testing.T) {
	// viper 语义：环境变量为空字符串视为未设置，回退默认值
	unsetAll(t)
	t.Setenv("F1GUIDE_HTTP_ADDR", "")

	cfg := Load()

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q（空环境变量应回退默认）", cfg.HTTPAddr, ":8080")
	}
}
