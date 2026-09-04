package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunReturnsErrorWithoutOIDCIssuer(t *testing.T) {
	var stderr bytes.Buffer
	code := run(nil, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "invalid server configuration") {
		t.Fatalf("stderr = %q, want invalid server configuration", stderr.String())
	}
}
