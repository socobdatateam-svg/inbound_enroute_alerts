package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsMissingValues(t *testing.T) {
	t.Setenv("SEATALK_APP_ID", "")

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("SEATALK_APP_ID=from-file\n"), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadDotEnv(envPath); err != nil {
		t.Fatalf("load dot env: %v", err)
	}
	if got := os.Getenv("SEATALK_APP_ID"); got != "from-file" {
		t.Fatalf("SEATALK_APP_ID = %q, want from-file", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingValues(t *testing.T) {
	t.Setenv("SEATALK_APP_ID", "from-env")

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("SEATALK_APP_ID=from-file\n"), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := loadDotEnv(envPath); err != nil {
		t.Fatalf("load dot env: %v", err)
	}
	if got := os.Getenv("SEATALK_APP_ID"); got != "from-env" {
		t.Fatalf("SEATALK_APP_ID = %q, want from-env", got)
	}
}

func TestLoadDotEnvIgnoresMissingFile(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), ".env")); err != nil {
		t.Fatalf("load missing dot env: %v", err)
	}
}
