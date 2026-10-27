package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantRepo string
		wantEnd  bool
		wantOK   bool
	}{
		{name: "single repository", path: "/repo-name", wantRepo: "repo-name", wantOK: true},
		{name: "nested repository", path: "/group/repo-name", wantRepo: "group/repo-name", wantOK: true},
		{name: "deep nested repository", path: "/team/project/repo-name", wantRepo: "team/project/repo-name", wantOK: true},
		{name: "info refs endpoint", path: "/group/repo-name/info/refs", wantRepo: "group/repo-name", wantEnd: true, wantOK: true},
		{name: "receive pack endpoint", path: "/group/repo-name/git-receive-pack", wantRepo: "group/repo-name", wantEnd: true, wantOK: true},
		{name: "upload pack endpoint", path: "/group/repo-name/git-upload-pack", wantRepo: "group/repo-name", wantEnd: true, wantOK: true},
		{name: "root is invalid", path: "/", wantOK: false},
		{name: "parent traversal is invalid", path: "/../repo-name", wantOK: false},
		{name: "dot path is invalid", path: "/./repo-name", wantOK: false},
		{name: "double slash invalid", path: "//group//repo", wantOK: false},
		{name: "backslash invalid", path: "/repo\\name", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, endpoint, ok := repositoryPath(tt.path)
			if ok != tt.wantOK {
				t.Fatalf("repositoryPath(%q) ok = %v, want %v", tt.path, ok, tt.wantOK)
			}
			if repo != tt.wantRepo {
				t.Fatalf("repositoryPath(%q) repo = %q, want %q", tt.path, repo, tt.wantRepo)
			}
			if endpoint != tt.wantEnd {
				t.Fatalf("repositoryPath(%q) endpoint = %v, want %v", tt.path, endpoint, tt.wantEnd)
			}
		})
	}
}

func TestGitRepositoryHandlerListsRepositories(t *testing.T) {
	root := t.TempDir()
	createBareRepo(t, filepath.Join(root, "group", "repo-one"))
	createBareRepo(t, filepath.Join(root, "repo-two"))

	handler := &gitRepositoryHandler{
		root:   root,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		backend: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var payload struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}

	if len(payload.Repositories) != 2 {
		t.Fatalf("repositories = %#v, want 2 entries", payload.Repositories)
	}
	if !contains(payload.Repositories, "group/repo-one") || !contains(payload.Repositories, "repo-two") {
		t.Fatalf("unexpected repository list: %#v", payload.Repositories)
	}
}

func TestGitRepositoryHandlerCreatesAndDeletesRepositories(t *testing.T) {
	root := t.TempDir()
	handler := &gitRepositoryHandler{
		root:   root,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		backend: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	if err := handler.createRepository(filepath.Join(root, "group", "repo-name")); err != nil {
		t.Fatalf("createRepository returned error: %v", err)
	}
	if !isBareRepository(filepath.Join(root, "group", "repo-name")) {
		t.Fatal("createRepository did not create a bare repository")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/group/repo-name", nil)
	deleteRes := httptest.NewRecorder()
	handler.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d", deleteRes.Code, http.StatusNoContent)
	}
	if _, err := os.Stat(filepath.Join(root, "group", "repo-name")); err == nil {
		t.Fatal("DELETE did not remove the repository")
	}
}

func TestGitRepositoryHandlerUsesTemplateHooksWhenCreatingRepository(t *testing.T) {
	root := t.TempDir()
	templateRoot := filepath.Join(root, "template")
	if err := os.MkdirAll(filepath.Join(templateRoot, "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir template hooks: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateRoot, "hooks", "pre-receive"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write template hook: %v", err)
	}

	handler := &gitRepositoryHandler{
		root:     root,
		template: templateRoot,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		backend: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	repoPath := filepath.Join(root, "group", "repo-name")
	if err := handler.createRepository(repoPath); err != nil {
		t.Fatalf("createRepository returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, "hooks", "pre-receive")); err != nil {
		t.Fatalf("template hook was not copied into repository: %v", err)
	}
}

func createBareRepo(t *testing.T, repoPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(repoPath), err)
	}
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("mkdir repo %s: %v", repoPath, err)
	}
	for _, name := range []string{"HEAD", "config", "objects", "refs"} {
		if err := os.MkdirAll(filepath.Join(repoPath, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Join(repoPath, name), err)
		}
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestRepositoryPathRejectsTraversalAndBadSegments(t *testing.T) {
	for _, pathValue := range []string{"/", "/../repo-name", "/./repo-name", "//group//repo", "/repo\\name", "/group/../repo-name"} {
		if repo, endpoint, ok := repositoryPath(pathValue); ok || repo != "" || endpoint {
			t.Fatalf("repositoryPath(%q) = (%q, %v, %v), want invalid", pathValue, repo, endpoint, ok)
		}
	}
}

func TestShouldCreateRepositoryOnlyOnReceivePack(t *testing.T) {
	if shouldCreateRepository(httptest.NewRequest(http.MethodPost, "/group/repo-name/git-upload-pack", nil), true) {
		t.Fatal("git-upload-pack should not create a repository")
	}
	if !shouldCreateRepository(httptest.NewRequest(http.MethodPost, "/group/repo-name/git-receive-pack?service=git-receive-pack", nil), true) {
		t.Fatal("git-receive-pack should create a repository")
	}
	if shouldCreateRepository(httptest.NewRequest(http.MethodGet, "/group/repo-name", nil), false) {
		t.Fatal("non-endpoint requests should not create a repository")
	}
}

func TestRepositoryPathAllowsNestedGroups(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/group/repo-name", want: "group/repo-name"},
		{path: "/team/project/repo-name/info/refs", want: "team/project/repo-name"},
	} {
		repo, endpoint, ok := repositoryPath(tc.path)
		if !ok || repo != tc.want || !strings.Contains(tc.path, repo) {
			t.Fatalf("repositoryPath(%q) = (%q, %v, %v), want repo %q", tc.path, repo, endpoint, ok, tc.want)
		}
	}
}

func TestHookDispatchScriptRunsMatchingHookFromArchivedRepo(t *testing.T) {
	root := t.TempDir()
	hookRepo := filepath.Join(root, "hooks-repo")
	if err := os.MkdirAll(filepath.Join(hookRepo, "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir hook repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hookRepo, "hooks", "pre-receive"), []byte("#!/bin/sh\nprintf '%s\\n' \"$HAYEL_HOOK_NAME\" > \"$HOOK_LOG\"\n"), 0o755); err != nil {
		t.Fatalf("write hook script: %v", err)
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

	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("get module root: %v", err)
	}
	dispatchPath := filepath.Join(root, "hayel-hook-dispatch")
	buildCmd := exec.Command("go", "build", "-o", dispatchPath, "./cmd/hayel-hook-dispatch")
	buildCmd.Dir = moduleRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build hook dispatcher: %v\n%s", err, out)
	}

	symlinkPath := filepath.Join(root, "pre-receive")
	if err := os.Symlink(dispatchPath, symlinkPath); err != nil {
		t.Fatalf("symlink dispatcher: %v", err)
	}

	logPath := filepath.Join(root, "hook-output.txt")
	cmd := exec.Command(symlinkPath)
	cmd.Env = append(os.Environ(), "HAYEL_HOOK_REPO_DIR="+hookRepo, "HOOK_LOG="+logPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("dispatch hook failed: %v\n%s", err, out)
	}
	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read hook output: %v", err)
	}
	if string(contents) != "pre-receive\n" {
		t.Fatalf("hook output = %q, want %q", string(contents), "pre-receive\n")
	}
}
