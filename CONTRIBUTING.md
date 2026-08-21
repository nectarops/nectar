<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Contributing to Dock-Weaver

Thank you for helping build a small, auditable Docker Swarm control plane.

## Before you start

Use Go 1.26, Node.js 24, pnpm 11, and Docker. Discuss large architecture or product changes in an issue before implementation. Report vulnerabilities through GitHub private security advisories, never through a public issue.

## Development

```bash
make install-web
make verify
```

Keep changes focused. Add regression tests for changed behavior, update the OpenAPI contract and documentation when applicable, and include screenshots for visible UI work. Do not commit credentials, private keys, Docker join tokens, registry secrets, generated build output, databases, or production logs.

## Commits and pull requests

Commits and squash-merge titles follow AngularJS/Conventional Commits:

```text
feat(web): add deployment progress view
fix(installer): preserve an existing swarm membership
```

Use the types and detailed rules in [AGENTS.md](AGENTS.md). A pull request must explain motivation and user impact, link relevant issues, include tests, and pass `make verify`. Keep unrelated behavior in separate pull requests.

By contributing, you agree that your contribution is licensed under `AGPL-3.0-only`.
