package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerLogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/group/repo-name", nil)
	res := httptest.NewRecorder()
	requestLogger(logger, next).ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusAccepted)
	}
	output := buf.String()
	if !strings.Contains(output, "request") || !strings.Contains(output, "/group/repo-name") {
		t.Fatalf("log output = %q, want method/path fields", output)
	}
}
