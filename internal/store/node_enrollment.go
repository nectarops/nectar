// SPDX-License-Identifier: AGPL-3.0-only

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nectarops/nectar/internal/domain"
)

func (s *Store) CreateNodeEnrollment(
	ctx context.Context,
	enrollment domain.NodeEnrollment,
	tokenHash []byte,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin node enrollment transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO node_enrollments(
			id, token_hash, requested_role, status, message, expires_at, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		enrollment.ID,
		tokenHash,
		enrollment.RequestedRole,
		enrollment.Status,
		enrollment.Message,
		enrollment.ExpiresAt.Unix(),
		enrollment.CreatedBy,
		enrollment.CreatedAt.Unix(),
		enrollment.UpdatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("create node enrollment: %w", err)
	}

	if err := insertNodeEnrollmentEvent(
		ctx,
		tx,
		enrollment.ID,
		enrollment.Status,
		"Enrollment command created",
		enrollment.CreatedAt,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit node enrollment transaction: %w", err)
	}
	return nil
}

func (s *Store) NodeEnrollmentByID(
	ctx context.Context,
	id string,
) (domain.NodeEnrollment, error) {
	row := s.db.QueryRowContext(ctx, nodeEnrollmentSelect+" WHERE id = ?", id)
	return scanNodeEnrollment(row)
}

func (s *Store) NodeEnrollmentByTokenHash(
	ctx context.Context,
	tokenHash []byte,
) (domain.NodeEnrollment, error) {
	row := s.db.QueryRowContext(ctx, nodeEnrollmentSelect+" WHERE token_hash = ?", tokenHash)
	enrollment, err := scanNodeEnrollment(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NodeEnrollment{}, domain.ErrInvalidEnrollmentToken
	}
	return enrollment, err
}

func (s *Store) ListNodeEnrollments(
	ctx context.Context,
	limit int,
) ([]domain.NodeEnrollment, error) {
	rows, err := s.db.QueryContext(
		ctx,
		nodeEnrollmentSelect+" ORDER BY created_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list node enrollments: %w", err)
	}
	defer rows.Close()

	enrollments := make([]domain.NodeEnrollment, 0)
	for rows.Next() {
		enrollment, scanErr := scanNodeEnrollment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		enrollments = append(enrollments, enrollment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node enrollments: %w", err)
	}
	return enrollments, nil
}

func (s *Store) ClaimNodeEnrollment(
	ctx context.Context,
	id string,
	claim domain.NodeEnrollmentClaim,
	now time.Time,
) (domain.NodeEnrollment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.NodeEnrollment{}, fmt.Errorf("begin claim node enrollment transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	enrollment, err := scanNodeEnrollment(tx.QueryRowContext(ctx, nodeEnrollmentSelect+" WHERE id = ?", id))
	if err != nil {
		return domain.NodeEnrollment{}, err
	}
	if !now.Before(enrollment.ExpiresAt) {
		if err := updateNodeEnrollmentStatus(
			ctx,
			tx,
			id,
			domain.NodeEnrollmentExpired,
			"Enrollment expired before the node joined",
			now,
		); err != nil {
			return domain.NodeEnrollment{}, err
		}
		if err := tx.Commit(); err != nil {
			return domain.NodeEnrollment{}, fmt.Errorf("commit expired node enrollment: %w", err)
		}
		return domain.NodeEnrollment{}, domain.ErrEnrollmentExpired
	}
	if enrollment.Terminal() {
		return domain.NodeEnrollment{}, domain.ErrEnrollmentTerminal
	}
	if enrollment.MachineIDHash != "" && enrollment.MachineIDHash != claim.MachineIDHash {
		return domain.NodeEnrollment{}, domain.ErrEnrollmentClaimed
	}
	if enrollment.MachineIDHash == claim.MachineIDHash && enrollment.Status != domain.NodeEnrollmentFailed {
		return enrollment, nil
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE node_enrollments SET
			status = ?, hostname = ?, machine_id_hash = ?, operating_system = ?, architecture = ?,
			advertise_address = ?, data_path_address = ?, docker_version = ?, message = ?, updated_at = ?
		 WHERE id = ? AND status NOT IN (?, ?, ?, ?)`,
		domain.NodeEnrollmentClaimed,
		claim.Hostname,
		claim.MachineIDHash,
		claim.OperatingSystem,
		claim.Architecture,
		claim.AdvertiseAddress,
		claim.DataPathAddress,
		claim.DockerVersion,
		"Node identity verified; preparing Docker",
		now.Unix(),
		id,
		domain.NodeEnrollmentCompleted,
		domain.NodeEnrollmentPromotionBlocked,
		domain.NodeEnrollmentRevoked,
		domain.NodeEnrollmentExpired,
	)
	if err != nil {
		return domain.NodeEnrollment{}, fmt.Errorf("claim node enrollment: %w", err)
	}
	if enrollment.Status == domain.NodeEnrollmentPending || enrollment.Status == domain.NodeEnrollmentFailed {
		if err := insertNodeEnrollmentEvent(
			ctx,
			tx,
			id,
			domain.NodeEnrollmentClaimed,
			"Node claimed the enrollment command",
			now,
		); err != nil {
			return domain.NodeEnrollment{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.NodeEnrollment{}, fmt.Errorf("commit claimed node enrollment: %w", err)
	}
	return s.NodeEnrollmentByID(ctx, id)
}

func (s *Store) RecordNodeEnrollmentProgress(
	ctx context.Context,
	id string,
	status domain.NodeEnrollmentStatus,
	message string,
	dockerVersion string,
	now time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin node enrollment progress transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(
		ctx,
		`UPDATE node_enrollments
		 SET status = ?, message = ?, docker_version = CASE WHEN ? = '' THEN docker_version ELSE ? END,
		     updated_at = ?
		 WHERE id = ? AND status NOT IN (?, ?, ?, ?)`,
		status,
		message,
		dockerVersion,
		dockerVersion,
		now.Unix(),
		id,
		domain.NodeEnrollmentCompleted,
		domain.NodeEnrollmentPromotionBlocked,
		domain.NodeEnrollmentRevoked,
		domain.NodeEnrollmentExpired,
	)
	if err != nil {
		return fmt.Errorf("record node enrollment progress: %w", err)
	}
	if err := requireChangedRow(result, "node enrollment"); err != nil {
		return err
	}
	if err := insertNodeEnrollmentEvent(ctx, tx, id, status, message, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit node enrollment progress: %w", err)
	}
	return nil
}

func (s *Store) FinishNodeEnrollment(
	ctx context.Context,
	id string,
	status domain.NodeEnrollmentStatus,
	nodeID string,
	dockerVersion string,
	message string,
	now time.Time,
) (domain.NodeEnrollment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.NodeEnrollment{}, fmt.Errorf("begin finished node enrollment transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(
		ctx,
		`UPDATE node_enrollments
		 SET status = ?, node_id = ?, docker_version = ?, message = ?, updated_at = ?
		 WHERE id = ? AND status NOT IN (?, ?, ?, ?)`,
		status,
		nodeID,
		dockerVersion,
		message,
		now.Unix(),
		id,
		domain.NodeEnrollmentCompleted,
		domain.NodeEnrollmentPromotionBlocked,
		domain.NodeEnrollmentRevoked,
		domain.NodeEnrollmentExpired,
	)
	if err != nil {
		return domain.NodeEnrollment{}, fmt.Errorf("finish node enrollment: %w", err)
	}
	if err := requireChangedRow(result, "node enrollment"); err != nil {
		return domain.NodeEnrollment{}, err
	}
	if err := insertNodeEnrollmentEvent(ctx, tx, id, status, message, now); err != nil {
		return domain.NodeEnrollment{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.NodeEnrollment{}, fmt.Errorf("commit finished node enrollment: %w", err)
	}
	return s.NodeEnrollmentByID(ctx, id)
}

func (s *Store) RevokeNodeEnrollment(
	ctx context.Context,
	id string,
	now time.Time,
) (domain.NodeEnrollment, error) {
	enrollment, err := s.NodeEnrollmentByID(ctx, id)
	if err != nil {
		return domain.NodeEnrollment{}, err
	}
	if enrollment.Terminal() {
		return domain.NodeEnrollment{}, domain.ErrEnrollmentTerminal
	}
	return s.FinishNodeEnrollment(
		ctx,
		id,
		domain.NodeEnrollmentRevoked,
		enrollment.NodeID,
		enrollment.DockerVersion,
		"Enrollment revoked by an Owner",
		now,
	)
}

func (s *Store) NodeEnrollmentEvents(
	ctx context.Context,
	enrollmentID string,
	afterID int64,
) ([]domain.NodeEnrollmentEvent, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, enrollment_id, status, message, created_at
		 FROM node_enrollment_events
		 WHERE enrollment_id = ? AND id > ?
		 ORDER BY id`,
		enrollmentID,
		afterID,
	)
	if err != nil {
		return nil, fmt.Errorf("list node enrollment events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.NodeEnrollmentEvent, 0)
	for rows.Next() {
		var event domain.NodeEnrollmentEvent
		var createdAt int64
		if err := rows.Scan(
			&event.ID,
			&event.EnrollmentID,
			&event.Status,
			&event.Message,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan node enrollment event: %w", err)
		}
		event.CreatedAt = time.Unix(createdAt, 0).UTC()
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node enrollment events: %w", err)
	}
	return events, nil
}

const nodeEnrollmentSelect = `SELECT
	id, requested_role, status, hostname, machine_id_hash, operating_system, architecture,
	advertise_address, data_path_address, docker_version, node_id, message,
	expires_at, created_by, created_at, updated_at
FROM node_enrollments`

type rowScanner interface {
	Scan(...any) error
}

func scanNodeEnrollment(row rowScanner) (domain.NodeEnrollment, error) {
	var enrollment domain.NodeEnrollment
	var expiresAt int64
	var createdAt int64
	var updatedAt int64
	err := row.Scan(
		&enrollment.ID,
		&enrollment.RequestedRole,
		&enrollment.Status,
		&enrollment.Hostname,
		&enrollment.MachineIDHash,
		&enrollment.OperatingSystem,
		&enrollment.Architecture,
		&enrollment.AdvertiseAddress,
		&enrollment.DataPathAddress,
		&enrollment.DockerVersion,
		&enrollment.NodeID,
		&enrollment.Message,
		&expiresAt,
		&enrollment.CreatedBy,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NodeEnrollment{}, sql.ErrNoRows
		}
		return domain.NodeEnrollment{}, fmt.Errorf("scan node enrollment: %w", err)
	}
	enrollment.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	enrollment.CreatedAt = time.Unix(createdAt, 0).UTC()
	enrollment.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return enrollment, nil
}

func insertNodeEnrollmentEvent(
	ctx context.Context,
	tx *sql.Tx,
	enrollmentID string,
	status domain.NodeEnrollmentStatus,
	message string,
	createdAt time.Time,
) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO node_enrollment_events(enrollment_id, status, message, created_at)
		 VALUES (?, ?, ?, ?)`,
		enrollmentID,
		status,
		message,
		createdAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("create node enrollment event: %w", err)
	}
	return nil
}

func updateNodeEnrollmentStatus(
	ctx context.Context,
	tx *sql.Tx,
	id string,
	status domain.NodeEnrollmentStatus,
	message string,
	now time.Time,
) error {
	_, err := tx.ExecContext(
		ctx,
		"UPDATE node_enrollments SET status = ?, message = ?, updated_at = ? WHERE id = ?",
		status,
		message,
		now.Unix(),
		id,
	)
	if err != nil {
		return fmt.Errorf("update node enrollment status: %w", err)
	}
	return insertNodeEnrollmentEvent(ctx, tx, id, status, message, now)
}

func requireChangedRow(result sql.Result, subject string) error {
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read changed %s rows: %w", subject, err)
	}
	if changed == 0 {
		return fmt.Errorf("%s was not found", subject)
	}
	return nil
}
