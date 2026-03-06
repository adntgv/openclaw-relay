package config

import (
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ClientConfig holds client configuration
type ClientConfig struct {
	URL          string        `yaml:"url"`
	Token        string        `yaml:"token"`
	ClawID       string        `yaml:"claw_id"`
	Capabilities []string      `yaml:"capabilities"`
	AllowedCmds  []string      `yaml:"allowed_commands"`
	Shell        ShellConfig   `yaml:"shell"`
	HooksDir     string        `yaml:"hooks_dir"`
}

// ShellConfig holds shell execution configuration
type ShellConfig struct {
	Timeout         time.Duration `yaml:"timeout"`
	AllowedBinaries []string      `yaml:"allowed_binaries"`
}

// Load loads config from file, overlaying with env vars
func Load(path string) (*ClientConfig, error) {
	cfg := &ClientConfig{
		Capabilities: []string{"shell"},
		Shell: ShellConfig{
			Timeout: 30 * time.Second,
		},
	}

	// Load from file if it exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Override with env vars
	if v := os.Getenv("RELAY_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("RELAY_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("RELAY_CLAW_ID"); v != "" {
		cfg.ClawID = v
	}
	if v := os.Getenv("RELAY_CAPABILITIES"); v != "" {
		cfg.Capabilities = strings.Split(v, ",")
	}

	return cfg, nil
}
