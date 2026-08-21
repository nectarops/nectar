# Dock-Weaver Repository Instructions

## Scope

These instructions apply to the entire repository.

Dock-Weaver is a free and open-source, self-hosted Docker Swarm management platform for small and medium-sized teams. The product must remain usable without a hosted account, node-count license, or paid feature gate for core cluster-management capabilities.

## Required Technology Stack

### Backend

- Language: Go.
- Distribution: a single production binary where practical.
- HTTP interface: versioned REST/JSON APIs under `/api/v1`.
- Long-running operation updates: Server-Sent Events (SSE).
- Static frontend assets: embed the production frontend build into the Go binary with `go:embed`.
- Docker integration: Docker Engine Go SDK; avoid parsing human-oriented CLI output when a structured API exists.
- SSH integration: Go SSH libraries, with known-hosts verification and redacted operation logs.
- Persistence for the MVP: SQLite in WAL mode with versioned schema migrations.
- Background work: durable, idempotent operations persisted in the database. Browser refreshes and API retries must not duplicate an installation or deployment.
- Logging: structured logs with request, operation, and release correlation IDs. Secrets must be redacted before they reach any logger.

### Web Frontend

- Framework: React with TypeScript. Do not introduce Vue.
- Build tool: Vite.
- Component system: shadcn/ui.
- Styling: Tailwind CSS and the primitives used by shadcn/ui.
- Accessibility: preserve keyboard navigation, visible focus states, semantic labels, and reduced-motion support.
- Data access: keep server state separate from local UI state; centralize API calls and error normalization.
- Real-time progress: consume SSE for node installation and release events.
- Tests: Vitest and React Testing Library for components; use Playwright for critical end-to-end flows once the UI is runnable.

### Packaging and Runtime

- Build Go and React in a reproducible multi-stage Docker build.
- Run Dock-Weaver as a Docker Swarm service constrained to a Manager node carrying the Dock-Weaver control label.
- Use Traefik's Swarm provider for application routing and automatic HTTPS.
- Keep runtime images minimal and run as a non-root user wherever Docker API access does not require elevated privileges.
- Pin direct dependency versions and commit lockfiles/checksums.

### Host Installer

- Provide a root-level `install.sh` that supports the public `curl -fsSL <release-url>/install.sh | sudo bash` installation flow.
- The same script must support non-interactive flags, including `--docker-version`, `--advertise-addr`, `--web-port`, `--dock-weaver-version`, and `--dry-run`.
- The installer must be idempotent and safe to rerun after interruption. Detect completed steps instead of blindly repeating destructive commands.
- Detect supported Linux distributions, package managers, architectures, existing Docker installations, existing Swarm membership, port conflicts, and ambiguous network interfaces before changing the host.
- Never silently replace, upgrade, or downgrade an existing Docker installation. Require an explicit override when the installed version conflicts with the requested version.
- Install Docker from official distribution-specific Docker repositories, then pin and verify the selected version.
- Initialize Swarm only when the host is not already a member. Never force a host out of an existing Swarm.
- Pull a pinned Dock-Weaver image, create required networks, volumes, configs, and secrets, deploy the service, and wait for its readiness endpoint.
- On success, print the exact Web setup URL and a one-time initialization token. The token must not be persisted in ordinary logs.
- Return a non-zero exit code on failure and print a redacted diagnostic plus a safe resume command.
- Verify checksums or signatures for every artifact downloaded by the bootstrap script. Publish checksums alongside release assets.
- Document a download-inspect-verify-run alternative to piping a remote script directly into a privileged shell.

## Expected Repository Layout

Use this layout when scaffolding the implementation unless a documented architectural decision supersedes it:

```text
cmd/dock-weaver/              minimal Go application entry point
internal/app/                 process wiring, lifecycle, and graceful shutdown
internal/api/                 HTTP handlers, middleware, and transport DTOs
internal/application/         use cases and operation orchestration
internal/domain/              domain models, errors, and invariants
internal/docker/              Docker Engine and Swarm adapters
internal/ssh/                 remote host inspection and installation
internal/store/               SQLite repositories
internal/store/migrations/    ordered, embedded database migrations
internal/security/            authentication, authorization, encryption, redaction
internal/traefik/             ingress and certificate configuration
internal/testutil/            shared Go test helpers when genuinely cross-package
api/                          OpenAPI contract and API examples
web/                          React, TypeScript, Vite, and shadcn/ui application
deploy/                       Swarm stacks, container, and bootstrap assets
scripts/                      development, verification, and release helpers
test/e2e/                     cross-component and installation tests
docs/                         product, architecture, operations, and security docs
.github/                      workflows, issue forms, and pull request templates
install.sh                    idempotent host bootstrap entry point
Dockerfile                    reproducible production image
Makefile                      canonical development and CI commands
go.mod / go.sum               Go module and dependency checksums
LICENSE                       AGPL-3.0 license text
README.md                     public project entry point
```

Keep domain and application logic independent from HTTP, SQLite, Docker, and SSH adapters. Do not put business logic directly in React components or HTTP handlers.

- All Go `main` packages must live under `cmd/<binary>/`; keep each `main.go` limited to flag parsing, dependency wiring, lifecycle setup, and calling the application runner.
- Use `internal/` for application code. Do not create `pkg/` unless the repository intentionally publishes a stable Go library for external consumers.
- Do not add generic `utils`, `helpers`, or `common` packages. Name packages after a focused domain or capability.
- Keep one Go module unless a concrete independent-versioning requirement justifies another module. Do not add `go.work` for a single-module repository.
- The `go.mod` module path must exactly match the final lowercase GitHub repository URL. Do not commit a placeholder module path.
- Co-locate Go unit tests with the code as `*_test.go`; use local `testdata/` directories for package fixtures and `test/e2e/` only for cross-component scenarios.
- Keep React features close to their routes and use shared components only after a pattern is repeated. Generated shadcn/ui primitives belong in `web/src/components/ui`.
- Keep generated files identifiable and reproducible. Never hand-edit generated output without also updating its source or generator.

## Code Quality Standards

### Go

- Write idiomatic Go and prefer clarity over cleverness. Keep the exported surface as small as possible.
- Format all Go code with `gofmt`; organize imports with `goimports`. Formatting output is not subject to personal preference.
- Break lines longer than roughly 120 characters at semantic boundaries. Calls with four or more arguments should place one argument per line.
- Use `:=` for initialized non-zero values and `var` when intentionally starting from the zero value.
- Initialize slices and maps when they are returned through APIs or written to; avoid surprising JSON `null` values and nil-map writes.
- Use named fields in struct literals outside tightly controlled same-package tests.
- Handle errors and edge cases first with early returns. Remove unnecessary `else` blocks after `return`, `break`, or `continue`.
- Wrap errors with useful operation context using `%w`; do not compare errors by matching their text.
- Put `context.Context` first in functions that perform I/O or can be canceled. Propagate cancellation and use bounded timeouts at adapter boundaries.
- Keep functions focused and normally at four parameters or fewer. Use an explicit parameter object when a coherent set of inputs would otherwise grow the signature.
- Prefer `switch` when repeatedly comparing the same value. Extract complex business conditions into meaningfully named booleans or domain methods.
- Restrict blank imports to visible registration points such as `main` or tests; do not use dot imports in production code.
- Avoid reflection and `any` when a concrete type or generic constraint expresses the contract.
- Do not introduce an interface before there are multiple implementations, an external boundary, or a demonstrated testing need. Define boundary interfaces from the consumer's perspective.
- Every goroutine must have a documented owner, cancellation path, and shutdown behavior. Prevent leaks and close owned resources deterministically.

### React and TypeScript

- Enable TypeScript strict mode. Do not use `any` to bypass type errors; use `unknown` plus validation at untrusted boundaries.
- Format and lint the frontend consistently with the checked-in project configuration. Do not suppress lint rules without a narrow explanation.
- Keep components focused on rendering and interaction. Put API access, validation, and reusable state transitions in dedicated modules or hooks.
- Validate API responses at the boundary when they contain security-sensitive or operational state; static TypeScript types do not validate runtime data.
- Preserve shadcn/ui composition and accessibility. New interactive controls require keyboard behavior, labels, focus handling, loading, empty, error, and disabled states.
- Do not expose secrets in client state, browser storage, URLs, analytics, error messages, or debug logs.
- Add component tests for meaningful behavior, not implementation details. Cover critical setup, node enrollment, deployment, and rollback flows with Playwright.

### Shell, SQL, and Configuration

- Shell scripts must use strict error handling appropriate to their shell, quote expansions, avoid `eval`, and pass `shellcheck`, `shfmt`, and syntax checks.
- Privileged shell operations must use validated values and explicit command arguments. Never concatenate user input into executable shell fragments.
- SQL migrations must be ordered, reviewable, and covered by upgrade tests. Parameterize application queries; never interpolate user input into SQL.
- Keep YAML, JSON, Dockerfiles, workflows, and Markdown formatted and lintable with repository-owned configuration.
- Comments should explain intent, invariants, security constraints, or surprising tradeoffs. Do not narrate obvious syntax or preserve dead code in comments.

## Required Quality Gates

Every pull request must pass the same commands locally and in GitHub Actions. The Makefile must provide stable entry points so CI does not contain a second, divergent build system.

Required checks once the corresponding code exists:

```text
format-check       verify Go, React, shell, Markdown, YAML, and JSON formatting
lint               run Go, TypeScript/React, shell, Dockerfile, and workflow linters
typecheck          run the TypeScript compiler without emitting files
test               run Go and frontend unit tests
test-race          run Go tests with the race detector
test-integration   exercise SQLite, Docker adapters, SSH adapters, and migrations
test-e2e           exercise the critical install/setup/deploy browser flows
build              build all Go commands, the React app, embedded assets, and image
vulnerability      run Go and JavaScript dependency vulnerability checks
license-check      detect incompatible dependencies and missing required notices
```

- At minimum, Go verification includes `go test ./...`, `go test -race ./...`, `go vet ./...`, the configured linter, and `go build ./cmd/...`.
- At minimum, Web verification includes a frozen-lockfile install, lint, TypeScript typecheck, unit tests, and production build.
- At minimum, installer verification includes `bash -n`, `shellcheck`, formatting validation, and isolated supported-distribution tests.
- Tests must be deterministic, hermetic where practical, and free of dependence on execution order. Network integration tests require explicit opt-in and bounded timeouts.
- Bug fixes require a regression test unless a test is technically infeasible; explain that exception in the pull request.
- Do not chase a coverage percentage by testing implementation trivia. Publish coverage, prevent unexplained regressions, and prioritize domain invariants, security boundaries, migrations, and failure recovery.
- CI must cancel superseded runs, use least-privilege permissions, pin third-party actions to immutable commit SHAs, cache only safe build inputs, and never expose secrets to untrusted pull requests.
- A change is not complete while relevant checks are skipped, flaky, or failing.

## GitHub Open-Source Repository Standards

The public repository must include and maintain:

```text
README.md
LICENSE
NOTICE                         when third-party attribution requires it
CONTRIBUTING.md
CODE_OF_CONDUCT.md
SECURITY.md
SUPPORT.md
GOVERNANCE.md
ROADMAP.md
CHANGELOG.md
.github/CODEOWNERS
.github/PULL_REQUEST_TEMPLATE.md
.github/ISSUE_TEMPLATE/bug.yml
.github/ISSUE_TEMPLATE/feature.yml
.github/ISSUE_TEMPLATE/config.yml
.github/dependabot.yml
.github/workflows/ci.yml
.github/workflows/codeql.yml
.github/workflows/release.yml
```

- README must show the project purpose, current maturity, supported platforms, installation, quick start, security warning, documentation, contribution path, and license.
- CONTRIBUTING must document prerequisites, setup, canonical Make targets, tests, commit format, pull request expectations, and how to report security issues.
- SECURITY must define supported versions, a private vulnerability-reporting channel, expected response process, and disclosure policy. Never request vulnerabilities through public issues.
- Keep issue forms actionable and request reproduction details without asking users to publish secrets or production credentials.
- Pull requests must be focused, explain motivation and behavior, link related issues, include tests, update docs, and provide screenshots or recordings for visible UI changes.
- Require review and successful status checks before merging the default branch. Do not force-push or commit directly to protected release branches.
- Use Semantic Versioning for releases. Publish immutable tags, release notes, checksums, an SBOM, container image digests, and source archives.
- Keep CHANGELOG user-focused. Generated release notes may assist but must not replace a reviewed summary of behavior, security, migrations, and breaking changes.
- Use automated dependency updates, CodeQL, secret scanning, dependency review, and release provenance when GitHub supports them for the repository.
- Do not commit credentials, private keys, production data, build outputs, editor state, local databases, coverage artifacts, or dependency directories.

## Commit Message Convention

All commits and squash-merge pull request titles must follow the AngularJS commit-message convention, compatible with Conventional Commits:

```text
<type>(<scope>)!: <subject>

<optional body>

<optional footer(s)>
```

Allowed types:

- `feat`: a user-visible feature.
- `fix`: a user-visible bug fix.
- `docs`: documentation-only changes.
- `style`: formatting or whitespace with no behavior change.
- `refactor`: code restructuring with no feature or bug-fix behavior.
- `perf`: a performance improvement.
- `test`: test additions or corrections.
- `build`: build system, packaging, or dependency changes.
- `ci`: continuous-integration and automation changes.
- `chore`: maintenance not covered by another type.
- `revert`: a revert of an earlier commit.

Use a short, stable scope when it adds clarity, such as `api`, `web`, `installer`, `docker`, `ssh`, `swarm`, `traefik`, `store`, `security`, `docs`, `ci`, or `deps`.

Subject rules:

- Write in English, imperative present tense, and lowercase after the colon.
- Describe what the commit does, not the implementation process.
- Do not end the subject with a period.
- Keep the complete header at 100 characters or fewer; prefer approximately 72 when it remains clear.

Body and footer rules:

- Use the body to explain motivation, user impact, important tradeoffs, and contrast with previous behavior. Wrap prose at approximately 100 characters.
- Mark breaking changes with `!` in the header and a `BREAKING CHANGE: <description>` footer.
- Reference issues with footers such as `Closes #123`, `Fixes #456`, or `Refs #789`.
- Do not mix unrelated changes in one commit. Each commit should build and pass relevant tests whenever practical.
- Never include generated-by marketing text, AI attribution, secrets, tokens, or internal credentials in commit messages.

Examples:

```text
feat(installer): add pinned Docker version selection
fix(swarm): preserve manager quorum during node upgrades
docs(readme): document checksum-verified installation
refactor(api): move deployment orchestration into application layer
feat(api)!: replace deployment event polling with SSE

BREAKING CHANGE: clients must consume the release event stream endpoint
Closes #142
```

## Engineering Requirements

- Prefer the smallest clear implementation over speculative abstractions.
- Validate all user-controlled hostnames, image references, versions, labels, paths, environment keys, and shell arguments.
- Never assemble privileged remote commands from unescaped user input.
- Treat Docker socket access as root-equivalent access.
- Do not persist or log Swarm join tokens. Retrieve them only for an active join operation and clear them after use.
- Encrypt stored SSH and registry credentials with an instance key supplied outside the database.
- Verify SSH host keys; never silently enable insecure host-key acceptance.
- Every node operation, deployment, upgrade, and rollback must have an idempotency key and an auditable result.
- Protect Manager quorum before drain, demotion, removal, restart, or Docker upgrade operations.
- Use health checks and bounded timeouts for Docker, SSH, registry, DNS, ACME, and downstream HTTP operations.
- Avoid using `latest` as an implicit image version. Record both the requested tag and resolved digest when available.
- Database migrations must be forward ordered, tested, and safe to rerun according to the selected migration tool's contract.
- Add tests with every behavior change. Favor unit tests for invariants and integration tests for adapter behavior.
- Update the product/technical plan when an architectural decision changes.

## UI Conventions

- Use shadcn/ui components from `web/src/components/ui` as the base design system; adapting generated component source is allowed and expected.
- Reuse shared form, table, status, confirmation, log-stream, and empty-state components instead of rebuilding them per page.
- Dangerous cluster actions must show the exact target and consequence and require explicit confirmation.
- Never display secret values after creation. Show metadata, fingerprints, and last-updated timestamps instead.
- Long operations must remain observable after navigation or refresh and expose clear retry/recovery actions.
- Keep basic workflows simple; place Swarm-specific controls behind advanced settings.
- The running Web UI must expose an easy-to-find source-code link and identify the running version/commit so operators can obtain the corresponding source required by AGPL network-interaction terms.

## License and Contributions

- The project is licensed under `AGPL-3.0-only`.
- New original source files should use `SPDX-License-Identifier: AGPL-3.0-only` in a comment format appropriate to the file type, except generated files or files whose format does not support comments.
- Dependencies and copied assets must have licenses compatible with AGPL-3.0-only and must retain required attribution notices.
- Do not add proprietary source, non-redistributable assets, telemetry that is enabled without informed consent, or code that requires a paid Dock-Weaver service for core operation.
- Keep copyright, trademark, and third-party notices accurate when distribution artifacts are produced.

## Verification

Before handing off an implementation change, run the relevant formatters, static checks, unit tests, and build steps for the changed Go and/or React code. When commands are added to the project, document the canonical commands in the README and keep CI aligned with them.
