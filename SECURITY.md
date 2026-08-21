<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Security Policy

## Supported versions

Until the first stable release, only the latest tagged alpha receives security fixes. Production use is not yet recommended.

## Reporting a vulnerability

Use **Security → Report a vulnerability** in this GitHub repository to open a private security advisory. Do not create a public issue. Include the affected version, impact, reproduction steps, and suggested mitigation when known. Remove real credentials, private keys, join tokens, and identifying production data.

Maintainers aim to acknowledge a report within 3 business days, provide an initial assessment within 7 business days, and coordinate a fix and disclosure based on severity. These are targets, not a service-level agreement.

## Operational security

Docker socket access is root-equivalent. Run Dock-Weaver only on a trusted Manager, restrict network access to setup and administration, use strong owner credentials, delete the bootstrap-token file after setup, back up SQLite securely, and keep Docker, Traefik, the host kernel, and Dock-Weaver patched.
