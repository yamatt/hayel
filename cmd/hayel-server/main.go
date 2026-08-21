// Command hayel-server runs the authenticated Git HTTP gateway.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/yamatt/hayel/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := server.ConfigFromFlags()
	if err != nil {
		logger.Error("could not load server configuration", "error", err)
		os.Exit(2)
	}
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid server configuration", "error", err)
		os.Exit(2)
	}

	service, err := server.New(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("could not create server", "error", err)
		os.Exit(1)
	}
	if err := service.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
