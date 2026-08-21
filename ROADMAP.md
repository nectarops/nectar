<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Roadmap

## 0.1 — first vertical slice

- Host installer, owner setup, authentication, embedded Web UI.
- Docker/Swarm inspection, Traefik bootstrap, and versioned image deployment.
- Reproducible image build, API contract, tests, and repository governance.

## 0.2 — safe node enrollment

- Verified SSH host fingerprints and encrypted credential storage.
- Durable, idempotent installation operations with SSE logs.
- Remote Docker version alignment, Worker/Manager join, labels, and quorum protection.

## 0.3 — release operations

- Persisted applications/releases, task health, progress streams, rollback, and audit events.
- Registry credentials, digest resolution, secrets/configs, and resource limits.
- Traefik configuration status, DNS/port preflight, and certificate diagnostics.

## 1.0 — production readiness

- Backup/restore, upgrade migrations, end-to-end installer tests, threat model, SBOM, signed releases, and documented recovery procedures.

Priorities may change after security findings and operator feedback. The roadmap is not a delivery promise.
