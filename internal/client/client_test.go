package client

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHandlerAllowlist(t *testing.T) {
	h := NewHandler([]string{"shell.exec"}, nil, 5*time.Second, "")

	// Allowed
	_, err := h.Execute("shell.exec", map[string]interface{}{"command": "echo hi"})
	if err != nil {
		t.Fatalf("allowed command failed: %v", err)
	}

	// Not allowed
	_, err = h.Execute("hook.run", map[string]interface{}{"name": "x"})
	if err == nil {
		t.Fatal("disallowed command should fail")
	}
}

func TestHandlerShellExec(t *testing.T) {
	h := NewHandler(nil, nil, 5*time.Second, "")

	result, err := h.Execute("shell.exec", map[string]interface{}{"command": "echo hello world"})
	if err != nil {
		t.Fatalf("shell.exec error: %v", err)
	}
	stdout, _ := result["stdout"].(string)
	if stdout != "hello world\n" {
		t.Errorf("stdout = %q, want 'hello world\\n'", stdout)
	}
	if result["exit_code"] != 0 {
		t.Errorf("exit_code = %v", result["exit_code"])
	}
}

func TestHandlerShellExecError(t *testing.T) {
	h := NewHandler(nil, nil, 5*time.Second, "")

	result, err := h.Execute("shell.exec", map[string]interface{}{"command": "exit 42"})
	if err != nil {
		t.Fatalf("shell.exec error: %v", err)
	}
	if result["exit_code"] != 42 {
		t.Errorf("exit_code = %v, want 42", result["exit_code"])
	}
}

func TestHandlerShellTimeout(t *testing.T) {
	h := NewHandler(nil, nil, 100*time.Millisecond, "")

	_, err := h.Execute("shell.exec", map[string]interface{}{"command": "sleep 10"})
	if err == nil {
		t.Fatal("should timeout")
	}
}

func TestHandlerHookRun(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "test-hook")
	os.WriteFile(script, []byte("#!/bin/sh\necho hook output"), 0755)

	h := NewHandler(nil, nil, 5*time.Second, dir)

	result, err := h.Execute("hook.run", map[string]interface{}{"name": "test-hook"})
	if err != nil {
		t.Fatalf("hook.run error: %v", err)
	}
	if result["stdout"] != "hook output\n" {
		t.Errorf("stdout = %q", result["stdout"])
	}
}

func TestHandlerHookPathTraversal(t *testing.T) {
	h := NewHandler(nil, nil, 5*time.Second, "/tmp")

	_, err := h.Execute("hook.run", map[string]interface{}{"name": "../etc/passwd"})
	if err == nil {
		t.Fatal("path traversal should be rejected")
	}
}

func TestBackoff(t *testing.T) {
	if d := backoff(0); d != 1*time.Second {
		t.Errorf("backoff(0) = %v", d)
	}
	if d := backoff(3); d != 8*time.Second {
		t.Errorf("backoff(3) = %v", d)
	}
	if d := backoff(100); d != maxBackoff {
		t.Errorf("backoff(100) = %v, want %v", d, maxBackoff)
	}
}
