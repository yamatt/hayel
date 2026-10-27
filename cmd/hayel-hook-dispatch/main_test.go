package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHookNameFromArgsUsesInvocationName(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"/usr/local/bin/hayel-hook-dispatch"}
	name, err := hookNameFromArgs()
	if err != nil {
		t.Fatalf("hookNameFromArgs() error = %v", err)
	}
	if name != "hayel-hook-dispatch" {
		t.Fatalf("hookNameFromArgs() = %q, want %q", name, "hayel-hook-dispatch")
	}
}

func TestHookNameFromArgsUsesSymlinkName(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"/tmp/pre-receive"}
	name, err := hookNameFromArgs()
	if err != nil {
		t.Fatalf("hookNameFromArgs() error = %v", err)
	}
	if name != "pre-receive" {
		t.Fatalf("hookNameFromArgs() = %q, want %q", name, "pre-receive")
	}
}

func TestRunExecutesMatchingHookFromArchivedRepo(t *testing.T) {
	root := t.TempDir()
	hookRepo := filepath.Join(root, "hook-template")
	if err := os.MkdirAll(filepath.Join(hookRepo, "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir hook repo: %v", err)
	}

	logPath := filepath.Join(root, "hook.log")
	hookScript := "#!/bin/sh\nprintf '%s %s %s\\n' \"$HAYEL_HOOK_NAME\" \"$1\" \"$2\" > \"$HOOK_LOG\"\n"
	if err := os.WriteFile(filepath.Join(hookRepo, "hooks", "pre-receive"), []byte(hookScript), 0o755); err != nil {
		t.Fatalf("write template hook: %v", err)
	}
	if out, err := exec.Command("git", "-C", hookRepo, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init hook repo: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", hookRepo, "config", "user.name", "Test User").CombinedOutput(); err != nil {
		t.Fatalf("git config user.name: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", hookRepo, "config", "user.email", "test@example.com").CombinedOutput(); err != nil {
		t.Fatalf("git config user.email: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", hookRepo, "add", "hooks/pre-receive").CombinedOutput(); err != nil {
		t.Fatalf("git add hook: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", hookRepo, "commit", "-m", "add hook").CombinedOutput(); err != nil {
		t.Fatalf("git commit hook: %v\n%s", err, out)
	}

	binPath := filepath.Join(root, "hayel-hook-dispatch")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/hayel-hook-dispatch")
	buildCmd.Dir = filepath.Join("..", "..")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build hook dispatcher: %v\n%s", err, out)
	}

	oldArgs := os.Args
	oldEnv := os.Getenv("HAYEL_HOOK_REPO_DIR")
	defer func() {
		os.Args = oldArgs
		if oldEnv == "" {
			_ = os.Unsetenv("HAYEL_HOOK_REPO_DIR")
		} else {
			_ = os.Setenv("HAYEL_HOOK_REPO_DIR", oldEnv)
		}
		_ = os.Unsetenv("HOOK_LOG")
	}()

	os.Args = []string{filepath.Join(root, "pre-receive"), "oldrev", "newrev"}
	if err := os.Setenv("HAYEL_HOOK_REPO_DIR", hookRepo); err != nil {
		t.Fatalf("set repo dir: %v", err)
	}
	if err := os.Setenv("HOOK_LOG", logPath); err != nil {
		t.Fatalf("set hook log: %v", err)
	}

	if code := run(); code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read hook log: %v", err)
	}
	if string(contents) != "pre-receive oldrev newrev\n" {
		t.Fatalf("hook log = %q, want %q", string(contents), "pre-receive oldrev newrev\n")
	}
}
