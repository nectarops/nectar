CREATE TABLE node_enrollments (
    id TEXT PRIMARY KEY,
    token_hash BLOB NOT NULL UNIQUE,
    requested_role TEXT NOT NULL CHECK (requested_role IN ('worker', 'manager')),
    status TEXT NOT NULL CHECK (status IN (
        'pending',
        'claimed',
        'installing',
        'joining',
        'verifying',
        'promoting',
        'completed',
        'promotion_blocked',
        'failed',
        'revoked',
        'expired'
    )),
    hostname TEXT NOT NULL DEFAULT '',
    machine_id_hash TEXT NOT NULL DEFAULT '',
    operating_system TEXT NOT NULL DEFAULT '',
    architecture TEXT NOT NULL DEFAULT '',
    advertise_address TEXT NOT NULL DEFAULT '',
    data_path_address TEXT NOT NULL DEFAULT '',
    docker_version TEXT NOT NULL DEFAULT '',
    node_id TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    expires_at INTEGER NOT NULL,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX node_enrollments_created_at_idx
    ON node_enrollments(created_at DESC);

CREATE INDEX node_enrollments_expires_at_idx
    ON node_enrollments(expires_at);

CREATE TABLE node_enrollment_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    enrollment_id TEXT NOT NULL REFERENCES node_enrollments(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX node_enrollment_events_enrollment_idx
    ON node_enrollment_events(enrollment_id, id);
