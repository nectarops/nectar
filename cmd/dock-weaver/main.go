// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ranen/dock-weaver/internal/app"
	"github.com/ranen/dock-weaver/internal/config"
	"github.com/ranen/dock-weaver/internal/webassets"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.Run(ctx, cfg, logger, webassets.Dist()); err != nil {
		logger.Error("run Dock-Weaver", "error", err)
		os.Exit(1)
	}
}
