package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	os.Exit(run())
}

func run() int {
	hookName, err := hookNameFromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	repoPath := os.Getenv("HAYEL_HOOK_REPO_DIR")
	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "HAYEL_HOOK_REPO_DIR not set; hook dispatch disabled")
		return 0
	}

	tmpDir, err := os.MkdirTemp("", "hayel-hook-dispatch-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmpDir)

	// Archive the current state of the hook repository
	// The repository locally is in a bare state. No files.
	// This makes a safe copy of the repository to be extracted to the temp directory.
	archive, err := archiveRepository(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "archive repository %q: %v\n", repoPath, err)
		return 1
	}

	tarCmd := exec.Command("tar", "-xf", "-", "-C", tmpDir)
	tarCmd.Stdin = bytes.NewReader(archive)
	if out, err := tarCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "extract archived hooks: %v\n%s", err, out)
		return 1
	}

	hookPath := filepath.Join(tmpDir, "hooks", hookName)
	if _, err := os.Stat(hookPath); err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "check hook %q: %v\n", hookPath, err)
		return 1
	}

	cmd := exec.Command(hookPath, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "HAYEL_HOOK_NAME="+hookName, "HAYEL_HOOK_REPO_DIR="+repoPath)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "run hook %q: %v\n", hookName, err)
		return 1
	}
	return 0
}

func hookNameFromArgs() (string, error) {
	name := filepath.Base(os.Args[0])
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "", fmt.Errorf("hook name could not be determined from invocation")
	}
	return name, nil
}

func archiveRepository(repoPath string) ([]byte, error) {
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		return exec.Command("git", "-C", repoPath, "archive", "--format=tar", "HEAD").Output()
	}
	return exec.Command("git", "--git-dir", repoPath, "archive", "--format=tar", "HEAD").Output()
}
