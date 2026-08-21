// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/ranen/dock-weaver/internal/api"
	"github.com/ranen/dock-weaver/internal/application"
	"github.com/ranen/dock-weaver/internal/config"
	dockeradapter "github.com/ranen/dock-weaver/internal/docker"
	"github.com/ranen/dock-weaver/internal/store"
)

func Run(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	assets fs.FS,
) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}

	database, err := store.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Warn("close database", "error", err)
		}
	}()

	configured, err := database.SetupCompleted(ctx)
	if err != nil {
		return err
	}
	if !configured && cfg.InitToken == "" {
		return errors.New("DW_INIT_TOKEN or DW_INIT_TOKEN_FILE is required before first setup")
	}

	authService, err := application.NewAuthService(
		database,
		cfg.InitToken,
		cfg.SessionDuration,
	)
	if err != nil {
		return err
	}

	inspector, inspectorErr := dockeradapter.NewInspector(cfg.DockerTimeout)
	var clusterReader application.ClusterReader
	var deploymentEngine application.DeploymentEngine
	var dockerReadiness api.Readiness
	if inspectorErr != nil {
		if cfg.RequireDocker {
			return inspectorErr
		}
		logger.Warn("Docker Engine is unavailable", "error", inspectorErr)
		unavailable := dockeradapter.NewUnavailableInspector(inspectorErr)
		clusterReader = unavailable
		deploymentEngine = unavailable
	} else {
		defer func() {
			if err := inspector.Close(); err != nil {
				logger.Warn("close Docker client", "error", err)
			}
		}()
		clusterReader = inspector
		deploymentEngine = inspector
		dockerReadiness = inspector
	}

	clusterService, err := application.NewClusterService(clusterReader)
	if err != nil {
		return err
	}
	deploymentService, err := application.NewDeploymentService(deploymentEngine)
	if err != nil {
		return err
	}

	apiServer, err := api.NewServer(api.Options{
		Logger:        logger,
		Auth:          authService,
		Cluster:       clusterService,
		Deployments:   deploymentService,
		Store:         database,
		Docker:        dockerReadiness,
		RequireDocker: cfg.RequireDocker,
		Assets:        assets,
		CookieSecure:  cfg.CookieSecure,
		SourceURL:     cfg.SourceURL,
		Version:       cfg.Version,
		Commit:        cfg.Commit,
	})
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.Address,
		Handler:           apiServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      3 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	serveErrors := make(chan error, 1)
	go func() {
		logger.Info("Dock-Weaver listening", "address", cfg.Address)
		serveErrors <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		return nil
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}
