package client

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const maxOutputSize = 64 * 1024 // 64KB

// Handler processes commands
type Handler struct {
	allowedCmds     map[string]bool
	allowedBinaries map[string]bool
	shellTimeout    time.Duration
	hooksDir        string
}

// NewHandler creates a new command handler
func NewHandler(allowedCmds, allowedBinaries []string, shellTimeout time.Duration, hooksDir string) *Handler {
	cmds := make(map[string]bool)
	for _, c := range allowedCmds {
		cmds[c] = true
	}
	bins := make(map[string]bool)
	for _, b := range allowedBinaries {
		bins[b] = true
	}
	if shellTimeout == 0 {
		shellTimeout = 30 * time.Second
	}
	return &Handler{
		allowedCmds:     cmds,
		allowedBinaries: bins,
		shellTimeout:    shellTimeout,
		hooksDir:        hooksDir,
	}
}

// Execute runs a command and returns the result
func (h *Handler) Execute(cmd string, args map[string]interface{}) (map[string]interface{}, error) {
	if len(h.allowedCmds) > 0 && !h.allowedCmds[cmd] {
		return nil, fmt.Errorf("command not allowed: %s", cmd)
	}

	switch cmd {
	case "hook.run":
		return h.hookRun(args)
	case "shell.exec":
		return h.shellExec(args)
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}
}

func (h *Handler) hookRun(args map[string]interface{}) (map[string]interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("hook name is required")
	}

	// Prevent path traversal
	if filepath.Base(name) != name {
		return nil, fmt.Errorf("invalid hook name")
	}

	hookPath := filepath.Join(h.hooksDir, name)
	if _, err := os.Stat(hookPath); err != nil {
		return nil, fmt.Errorf("hook not found: %s", name)
	}

	cmd := exec.Command(hookPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := truncate(stdout.String(), maxOutputSize)
	errOutput := truncate(stderr.String(), maxOutputSize)

	result := map[string]interface{}{
		"stdout":    output,
		"stderr":    errOutput,
		"exit_code": 0,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}
	return result, nil
}

func (h *Handler) shellExec(args map[string]interface{}) (map[string]interface{}, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Execute with timeout
	cmd := exec.Command("sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		result := map[string]interface{}{
			"stdout":    truncate(stdout.String(), maxOutputSize),
			"stderr":    truncate(stderr.String(), maxOutputSize),
			"exit_code": 0,
		}
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result["exit_code"] = exitErr.ExitCode()
			} else {
				return nil, err
			}
		}
		return result, nil
	case <-time.After(h.shellTimeout):
		cmd.Process.Kill()
		return nil, fmt.Errorf("command timed out after %s", h.shellTimeout)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "...(truncated)"
	}
	return s
}
