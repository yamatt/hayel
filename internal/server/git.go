package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// newGitHTTPHandler wraps Git's built-in git-http-backend CGI program.
func newGitHTTPHandler(repositoryRoot string, logger *slog.Logger) (http.Handler, error) {
	output, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		return nil, fmt.Errorf("find Git exec path: %w", err)
	}
	backendPath := filepath.Join(strings.TrimSpace(string(output)), "git-http-backend")
	backend := &cgi.Handler{
		Path: backendPath,
		Env: []string{
			"GIT_PROJECT_ROOT=" + repositoryRoot,
			"GIT_HTTP_EXPORT_ALL=1",
			"REMOTE_USER=oauth2",
		},
	}
	return &gitRepositoryHandler{root: repositoryRoot, backend: backend, logger: logger}, nil
}

type gitRepositoryHandler struct {
	root    string
	backend http.Handler
	logger  *slog.Logger
}

func (h *gitRepositoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.listRepositories(w)
		return
	}
	repository, endpoint, ok := repositoryPath(r.URL.Path)
	if !ok {
		http.Error(w, "invalid repository path", http.StatusBadRequest)
		return
	}
	fullPath := filepath.Join(h.root, filepath.FromSlash(repository))

	if r.Method == http.MethodDelete {
		if endpoint {
			http.Error(w, "DELETE must target the repository path", http.StatusBadRequest)
			return
		}
		if err := h.deleteRepository(fullPath); err != nil {
			h.logger.Error("could not delete repository", "repository", repository, "error", err)
			http.Error(w, "could not delete repository", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if shouldCreateRepository(r, endpoint) {
		if err := h.createRepository(fullPath); err != nil {
			h.logger.Error("could not create repository", "repository", repository, "error", err)
			http.Error(w, "could not create repository", http.StatusInternalServerError)
			return
		}
	}
	h.backend.ServeHTTP(w, r)
}

func (h *gitRepositoryHandler) listRepositories(w http.ResponseWriter) {
	repositories, err := h.repositories()
	if err != nil {
		h.logger.Error("could not list repositories", "error", err)
		http.Error(w, "could not list repositories", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Repositories []string `json:"repositories"`
	}{Repositories: repositories}); err != nil {
		h.logger.Error("could not encode repository list", "error", err)
	}
}

func (h *gitRepositoryHandler) repositories() ([]string, error) {
	repositories := make([]string, 0)
	err := filepath.WalkDir(h.root, func(current string, entry fs.DirEntry, err error) error {
		// Handle permission errors or missing path errors gracefully
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				if entry != nil && entry.IsDir() {
					return fs.SkipDir // Skip unreadable directories like lost+found
				}
				return nil // Skip unreadable files
			}
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err // Surface genuine system errors
		}

		if !entry.IsDir() || current == h.root {
			return nil
		}

		if isBareRepository(current) {
			relative, err := filepath.Rel(h.root, current)
			if err != nil {
				return err
			}
			repositories = append(repositories, filepath.ToSlash(relative))
			return fs.SkipDir
		}
		return nil
	})
	return repositories, err
}

func (h *gitRepositoryHandler) createRepository(repository string) error {
	if _, err := os.Stat(repository); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(repository), 0o755); err != nil {
		return err
	}
	return exec.Command("git", "init", "--bare", "--quiet", repository).Run()
}

func (h *gitRepositoryHandler) deleteRepository(repository string) error {
	info, err := os.Stat(repository)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory")
	}
	if !isBareRepository(repository) {
		return fmt.Errorf("not a bare Git repository")
	}
	return os.RemoveAll(repository)
}

func isBareRepository(repository string) bool {
	for _, required := range []string{"HEAD", "config", "objects", "refs"} {
		if _, err := os.Stat(filepath.Join(repository, required)); err != nil {
			return false
		}
	}
	return true
}

func repositoryPath(requestPath string) (repository string, endpoint bool, ok bool) {
	if requestPath == "" || requestPath == "/" {
		return "", false, false
	}
	if strings.Contains(requestPath, "\\") {
		return "", false, false
	}

	trimmed := strings.Trim(requestPath, "/")
	if trimmed == "" {
		return "", false, false
	}
	segments := strings.Split(trimmed, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", false, false
		}
	}

	clean := path.Clean("/" + requestPath)
	trimmed = strings.Trim(clean, "/")
	if trimmed == "" || trimmed == "." {
		return "", false, false
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) >= 2 && parts[len(parts)-2] == "info" && parts[len(parts)-1] == "refs" {
		repository = strings.Join(parts[:len(parts)-2], "/")
		endpoint = true
		return repository, endpoint, true
	}
	if len(parts) >= 2 && (parts[len(parts)-1] == "git-upload-pack" || parts[len(parts)-1] == "git-receive-pack") {
		repository = strings.Join(parts[:len(parts)-1], "/")
		endpoint = true
		return repository, endpoint, true
	}
	if len(parts) >= 1 {
		repository = strings.Join(parts, "/")
		return repository, false, true
	}

	return "", false, false
}

func shouldCreateRepository(r *http.Request, endpoint bool) bool {
	if !endpoint {
		return false
	}
	return r.URL.Query().Get("service") == "git-receive-pack" ||
		strings.HasSuffix(r.URL.Path, "/git-receive-pack")
}
