// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nectarops/nectar/internal/domain"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping SQLite database: %w", err)
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ready(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)`,
	); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := s.applyMigration(ctx, entry.Name()); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) applyMigration(ctx context.Context, name string) error {
	var applied int
	err := s.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
		name,
	).Scan(&applied)
	if err != nil {
		return fmt.Errorf("check migration %q: %w", name, err)
	}
	if applied > 0 {
		return nil
	}

	content, err := migrations.ReadFile("migrations/" + name)
	if err != nil {
		return fmt.Errorf("read migration %q: %w", name, err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %q: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("execute migration %q: %w", name, err)
	}
	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)",
		name,
		time.Now().UTC().Unix(),
	); err != nil {
		return fmt.Errorf("record migration %q: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", name, err)
	}

	return nil
}

func (s *Store) EnsureDesiredDockerVersion(ctx context.Context, version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return errors.New("desired Docker version is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Docker version policy transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO settings(key, value, updated_at) VALUES ('desired_docker_version', ?, ?)
		 ON CONFLICT(key) DO NOTHING`,
		version,
		time.Now().UTC().Unix(),
	); err != nil {
		return fmt.Errorf("initialize desired Docker version: %w", err)
	}

	var recorded string
	if err := tx.QueryRowContext(
		ctx,
		"SELECT value FROM settings WHERE key = 'desired_docker_version'",
	).Scan(&recorded); err != nil {
		return fmt.Errorf("read desired Docker version: %w", err)
	}
	if recorded != version {
		return fmt.Errorf(
			"%w: recorded %q, Manager reports %q",
			domain.ErrDockerVersionConflict,
			recorded,
			version,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Docker version policy: %w", err)
	}
	return nil
}

func (s *Store) DesiredDockerVersion(ctx context.Context) (string, error) {
	var version string
	err := s.db.QueryRowContext(
		ctx,
		"SELECT value FROM settings WHERE key = 'desired_docker_version'",
	).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read desired Docker version: %w", err)
	}
	return version, nil
}

func (s *Store) SetupCompleted(ctx context.Context) (bool, error) {
	var value string
	err := s.db.QueryRowContext(
		ctx,
		"SELECT value FROM settings WHERE key = 'setup_completed'",
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read setup status: %w", err)
	}

	return value == "true", nil
}

func (s *Store) ManagementAccess(ctx context.Context) (domain.ManagementAccess, error) {
	var access domain.ManagementAccess
	settings := []struct {
		key    string
		target *string
	}{
		{key: "management_domain", target: &access.Domain},
		{key: "acme_email", target: &access.ACMEEmail},
	}
	for _, setting := range settings {
		err := s.db.QueryRowContext(
			ctx,
			"SELECT value FROM settings WHERE key = ?",
			setting.key,
		).Scan(setting.target)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return domain.ManagementAccess{}, fmt.Errorf("read %s setting: %w", setting.key, err)
		}
	}
	return access, nil
}

func (s *Store) SaveManagementAccess(ctx context.Context, access domain.ManagementAccess) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin management access transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Unix()
	for key, value := range map[string]string{
		"management_domain": access.Domain,
		"acme_email":        access.ACMEEmail,
	} {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO settings(key, value, updated_at) VALUES (?, ?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			key,
			value,
			now,
		); err != nil {
			return fmt.Errorf("store %s setting: %w", key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit management access transaction: %w", err)
	}
	return nil
}

func (s *Store) CompleteSetup(
	ctx context.Context,
	username string,
	passwordHash string,
) (domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin setup transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var completed int
	if err := tx.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM settings WHERE key = 'setup_completed' AND value = 'true'",
	).Scan(&completed); err != nil {
		return domain.User{}, fmt.Errorf("check setup transaction: %w", err)
	}
	if completed > 0 {
		return domain.User{}, domain.ErrAlreadyConfigured
	}

	now := time.Now().UTC()
	result, err := tx.ExecContext(
		ctx,
		"INSERT INTO users(username, password_hash, role, created_at) VALUES (?, ?, 'owner', ?)",
		username,
		passwordHash,
		now.Unix(),
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("create owner account: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return domain.User{}, fmt.Errorf("read owner account ID: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO settings(key, value, updated_at) VALUES ('setup_completed', 'true', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		now.Unix(),
	); err != nil {
		return domain.User{}, fmt.Errorf("mark setup complete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.User{}, fmt.Errorf("commit setup transaction: %w", err)
	}

	return domain.User{
		ID:        userID,
		Username:  username,
		Role:      "owner",
		CreatedAt: now,
	}, nil
}

func (s *Store) UserByUsername(ctx context.Context, username string) (domain.StoredUser, error) {
	var user domain.StoredUser
	var createdAt int64
	err := s.db.QueryRowContext(
		ctx,
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StoredUser{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return domain.StoredUser{}, fmt.Errorf("find user: %w", err)
	}
	user.CreatedAt = time.Unix(createdAt, 0).UTC()

	return user, nil
}

func (s *Store) CreateSession(
	ctx context.Context,
	userID int64,
	tokenHash []byte,
	expiresAt time.Time,
) error {
	now := time.Now().UTC().Unix()
	if _, err := s.db.ExecContext(
		ctx,
		"INSERT INTO sessions(user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?)",
		userID,
		tokenHash,
		expiresAt.UTC().Unix(),
		now,
	); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (s *Store) UserBySession(ctx context.Context, tokenHash []byte) (domain.User, error) {
	var user domain.User
	var createdAt int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT users.id, users.username, users.role, users.created_at
		 FROM sessions
		 JOIN users ON users.id = sessions.user_id
		 WHERE sessions.token_hash = ? AND sessions.expires_at > ?`,
		tokenHash,
		time.Now().UTC().Unix(),
	).Scan(
		&user.ID,
		&user.Username,
		&user.Role,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("authenticate session: %w", err)
	}
	user.CreatedAt = time.Unix(createdAt, 0).UTC()

	return user, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if _, err := s.db.ExecContext(
		ctx,
		"DELETE FROM sessions WHERE token_hash = ?",
		tokenHash,
	); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	if _, err := s.db.ExecContext(
		ctx,
		"DELETE FROM sessions WHERE expires_at <= ?",
		time.Now().UTC().Unix(),
	); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}

	return nil
}
