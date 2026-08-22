// SPDX-License-Identifier: AGPL-3.0-only

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nectarops/nectar/internal/version"
)

const (
	defaultAddress         = ":8080"
	defaultDataDir         = "/var/lib/nectar"
	defaultSessionDuration = 24 * time.Hour
	defaultDockerTimeout   = 8 * time.Second
	defaultShutdownTimeout = 15 * time.Second
)

type Config struct {
	Address         string
	DataDir         string
	DatabasePath    string
	InitToken       string
	CookieSecure    bool
	RequireDocker   bool
	SessionDuration time.Duration
	DockerTimeout   time.Duration
	ShutdownTimeout time.Duration
	SourceURL       string
	Version         string
	Commit          string
}

func Load() (Config, error) {
	dataDir := valueOrDefault("NECTAR_DATA_DIR", defaultDataDir)
	initToken, err := loadInitToken()
	if err != nil {
		return Config{}, err
	}

	cookieSecure, err := boolValue("NECTAR_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	requireDocker, err := boolValue("NECTAR_REQUIRE_DOCKER", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Address:         valueOrDefault("NECTAR_ADDR", defaultAddress),
		DataDir:         dataDir,
		DatabasePath:    filepath.Join(dataDir, "nectar.db"),
		InitToken:       initToken,
		CookieSecure:    cookieSecure,
		RequireDocker:   requireDocker,
		SessionDuration: defaultSessionDuration,
		DockerTimeout:   defaultDockerTimeout,
		ShutdownTimeout: defaultShutdownTimeout,
		SourceURL:       valueOrDefault("NECTAR_SOURCE_URL", "https://github.com/nectarops/nectar"),
		Version:         valueOrDefault("NECTAR_VERSION", version.Version),
		Commit:          valueOrDefault("NECTAR_COMMIT", version.Commit),
	}, nil
}

func loadInitToken() (string, error) {
	if token := strings.TrimSpace(os.Getenv("NECTAR_INIT_TOKEN")); token != "" {
		return token, nil
	}

	path := strings.TrimSpace(os.Getenv("NECTAR_INIT_TOKEN_FILE"))
	if path == "" {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read initialization token file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

func boolValue(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}

func valueOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	return value
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Address) == "" {
		return errors.New("server address is required")
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return errors.New("data directory is required")
	}
	if c.SessionDuration <= 0 {
		return errors.New("session duration must be positive")
	}
	if c.DockerTimeout <= 0 {
		return errors.New("Docker timeout must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}

	return nil
}
