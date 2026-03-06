package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	os.WriteFile(path, []byte(`
url: ws://localhost:8080/ws
claw_id: test-claw
token: test-token
capabilities:
  - shell
  - browser
allowed_commands:
  - hook.run
  - shell.exec
shell:
  allowed_binaries:
    - /usr/bin/git
`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.ClawID != "test-claw" {
		t.Errorf("ClawID = %v", cfg.ClawID)
	}
	if len(cfg.Capabilities) != 2 {
		t.Errorf("Capabilities = %v", cfg.Capabilities)
	}
	if len(cfg.AllowedCmds) != 2 {
		t.Errorf("AllowedCmds = %v", cfg.AllowedCmds)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	os.Setenv("RELAY_URL", "ws://env:9090/ws")
	os.Setenv("RELAY_CLAW_ID", "env-claw")
	defer os.Unsetenv("RELAY_URL")
	defer os.Unsetenv("RELAY_CLAW_ID")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.URL != "ws://env:9090/ws" {
		t.Errorf("URL = %v", cfg.URL)
	}
	if cfg.ClawID != "env-claw" {
		t.Errorf("ClawID = %v", cfg.ClawID)
	}
}
