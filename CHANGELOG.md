<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Changelog

All notable user-facing changes are documented here. This project follows Semantic Versioning and the Keep a Changelog structure.

## [Unreleased]

### Added

- Go control-plane service with embedded React/shadcn Web UI.
- Owner bootstrap, Argon2id authentication, SQLite sessions, and security middleware.
- Docker Engine and Swarm overview using the Moby client.
- Versioned Swarm service deployment with Traefik and Let's Encrypt HTTP-01.
- Ubuntu/Debian installer with explicit Docker version safety checks.
- Reproducible container build, API contract, tests, and open-source project files.
- One-command local launcher with an isolated data directory and automatic browser opening.

### Fixed

- Prevent Nectar Swarm tasks from being rejected on hosts whose existing Docker networks overlap Swarm's
  automatically allocated Overlay subnet.
- Publish the Nectar Web port in host mode so an existing conflicting Swarm ingress network cannot reject the
  control-plane task.
