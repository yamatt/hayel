// Command hayel-server runs the authenticated Git HTTP gateway.
package main

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/yamatt/hayel/internal/server"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	oldArgs := os.Args
	os.Args = append([]string{"hayel-server"}, args...)
	defer func() { os.Args = oldArgs }()

	logger := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := server.ConfigFromFlags()
	if err != nil {
		logger.Error("could not load server configuration", "error", err)
		return 2
	}
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid server configuration", "error", err)
		return 2
	}

	service, err := server.New(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("could not create server", "error", err)
		return 1
	}
	if err := service.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		return 1
	}
	return 0
}
