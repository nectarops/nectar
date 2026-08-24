<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Nectar

Nectar is a free, self-hosted Docker Swarm control plane for small and medium-sized teams. It provides a focused Web interface for initializing an owner account, inspecting a real Swarm Manager, and deploying versioned container images behind Traefik with automatic Let's Encrypt HTTPS.

There is no hosted account, node-count license, telemetry requirement, or paid feature gate. The backend is Go; the embedded Web application is React, TypeScript, Vite, Tailwind CSS, and shadcn/ui.

> **Project status:** early alpha. The install/setup/cluster-inspection/deployment vertical slice is implemented. Remote SSH node enrollment, durable background operation streams, registry credentials, and rollback controls remain on the roadmap. Do not treat this release as a mature production control plane yet.

## What works today

- A single idempotent Ubuntu, Debian, and CentOS Stream host installer with explicit Docker Engine version
  selection.
- Safe handling of an existing Docker installation: version changes require `--force-docker-version`.
- Persistence of the verified Manager Docker Engine version as a protected cluster target in SQLite.
- Safe merging and validation of bounded Docker `json-file` log rotation in `/etc/docker/daemon.json`.
- Swarm initialization without forcing a host out of an existing cluster.
- On-demand Traefik `v3.7.1` installation after an Owner configures HTTPS access or deploys a domain.
- One Traefik replica pinned to the Nectar Manager, with host-mode ports 80 and 443 only on that node.
- A one-time bootstrap token and Argon2id-protected owner account.
- SQLite persistence in WAL mode with embedded, ordered migrations.
- HttpOnly session cookies, strict JSON decoding, same-origin mutation checks, and security headers.
- Live Docker Engine and Swarm status through the official Moby Go client.
- Versioned image deployment through the Docker Engine API; implicit `latest` is rejected.
- Post-login management-domain setup with Traefik routing and Let's Encrypt HTTP-01 HTTPS.
- HTTP-to-HTTPS redirects and Let's Encrypt HTTP-01 certificates for deployed domains.
- Start-first rolling updates with Docker-native rollback on failure.
- A reproducible multi-stage container build and an embedded single-binary Web application.

## Supported platforms

The host installer supports Ubuntu and Debian with `apt`, plus CentOS Stream 9 and 10 with `dnf`. All
supported distributions require systemd and may run on `amd64` or `arm64`. The installer accepts a new
host, an existing Swarm Manager, or an existing compatible Docker Engine. CentOS Linux 7 and 8 are not
supported because Docker no longer lists them as maintained installation targets.

Nectar must run on a Swarm Manager with access to `/var/run/docker.sock`. Docker socket access is equivalent to root access; isolate the control-plane host and restrict access to Nectar.

## Installation

No public container image or GitHub release is assumed to exist until the repository publishes `v0.1.0`. For a published release, the intended one-line flow is:

```bash
curl -fsSL https://github.com/nectarops/nectar/releases/download/v0.1.0/install.sh \
  | sudo bash -s -- \
      --docker-version 29.0.1 \
      --advertise-addr 192.0.2.10 \
      --nectar-version 0.1.0
```

The safer inspect-and-verify flow is:

```bash
curl -fLO https://github.com/nectarops/nectar/releases/download/v0.1.0/install.sh
curl -fLO https://github.com/nectarops/nectar/releases/download/v0.1.0/SHA256SUMS
sha256sum --check --ignore-missing SHA256SUMS
less install.sh
sudo bash install.sh --dry-run --advertise-addr 192.0.2.10
sudo bash install.sh --advertise-addr 192.0.2.10
```

Run `bash install.sh --help` for all options. The installer preserves unrelated Docker daemon settings, applies this log policy, validates the merged JSON with `dockerd`, and restarts Docker only when the file changed:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3",
    "compress": "true"
  }
}
```

For hosts that cannot reach Docker's default repository, set a trusted HTTPS mirror with the same Docker CE repository layout. The signing key fingerprint is still verified before the mirror is configured. For example, on Tencent Cloud:

```bash
sudo env NECTAR_DOCKER_REPOSITORY_URL=https://mirrors.cloud.tencent.com/docker-ce/linux \
  bash install.sh --advertise-addr 10.0.0.7
```

The installer then initializes Swarm when needed, deploys Nectar on the configured Web port, and prints the exact Web URL and one-time setup token after readiness succeeds. It does not create `traefik-public`, install Traefik, or occupy ports 80 and 443. The verified Docker Engine server version is stored in SQLite as `desired_docker_version`; rerunning the installer cannot silently replace that cluster policy. Delete the root-readable resume copy at `/var/lib/nectar/bootstrap-token` after setup. Docker logging defaults apply to containers created after the daemon restart; existing containers keep their original logging configuration.

On the first visit, open the printed `http://<manager-ip>:<port>` URL and create the Owner account. After signing in, open **HTTPS access** in the sidebar, enter the management domain and Let's Encrypt email, and submit the form. Point the domain to this Manager and allow inbound TCP 80 and 443 first. Nectar then creates the ingress network, installs or reconciles one Traefik replica on the same labeled Manager as Nectar, publishes 80 and 443 in host mode only on that node, and attaches an HTTPS route to its own Swarm service. Until this form is submitted, Traefik remains uninstalled and those ports remain unused by Nectar. The IP-and-port URL remains available as a recovery path; sign in again when opening the domain because browser cookies are scoped to each host.

To test an unpublished image, build and push it under a pinned tag, then set `NECTAR_IMAGE`:

```bash
sudo NECTAR_IMAGE=registry.example.com/ops/nectar:0.1.0 \
  bash install.sh --advertise-addr 192.0.2.10
```

## Docker version policy and additional nodes

The first Manager's verified Docker Engine server version becomes the cluster-wide target. The overview page shows both the Manager's live version and the SQLite target and warns if they differ.

Remote SSH node enrollment is not implemented in the current alpha. Its required version flow is:

1. Read `desired_docker_version` from the Manager's SQLite-backed cluster policy.
2. Inspect the candidate node's distribution, architecture, and existing Docker Engine.
3. If Docker is absent, verify that the official Docker repository provides the exact target Engine version, install its matching distribution package, and confirm the daemon reports the target version.
4. If Docker is already installed, verify that its daemon is healthy, preserve the existing version, and continue even when it differs from the target.
5. Record and display version drift instead of silently upgrading, downgrading, or blocking the existing installation.
6. Retrieve a short-lived Swarm join token, join the node, verify it is Ready, and clear the token without storing or logging it.

Until the node-enrollment workflow lands, do not copy the Manager installer onto a Worker: `install.sh` initializes or preserves a Manager and deploys the Nectar control plane. When adding a Worker manually, install **Cluster Docker target** only if Docker is absent; preserve an existing healthy Docker installation.

## Deploying an application

Open the Web UI, sign in, and use **Deploy a service**. Supply a service name, image repository, explicit version, internal container port, replica count, public domain, and Let's Encrypt email.

Before deploying, ensure that:

- The domain's A/AAAA record resolves to an address that reaches the Swarm.
- Inbound TCP ports 80 and 443 are open.
- The image is reachable from every node that might run the service.
- The application listens on the container port entered in the form.

The **HTTPS access** page creates Traefik and the `traefik-public` network on demand. An application deployment can also create them when needed. Later submissions with the same service name perform a rolling service update.

## Local development

Prerequisites are Go 1.26, Node.js 24, pnpm 11, and Docker for integration testing.

For a one-command local preview, run:

```bash
make dev
```

The development launcher installs the pinned web dependencies, builds the embedded React application and Go binary, keeps its SQLite database and bootstrap token under the project `.data` directory, starts the server at `http://127.0.0.1:8080`, and opens the page in the default browser. The same owner and session data are reused on later starts. Docker is optional for UI development; the cluster panel reports Docker as unavailable when the Engine is not running. Press `Ctrl+C` in the terminal running `make dev` to stop it.

Use `./scripts/dev.sh --help` to select another address or data directory, require Docker, reuse an existing build, or disable automatic browser opening.

`install.sh` changes a Linux host and cannot be exercised faithfully by `make dev` on macOS. Use
`sudo bash install.sh --dry-run --advertise-addr <linux-host-ip>` in a disposable Ubuntu, Debian, or CentOS
Stream VM first, then run it without `--dry-run` for a real installation.

Run all project checks:

```bash
make install-web
make verify
```

Run the frontend development server:

```bash
npm --prefix web run dev
```

Build the embedded production binary:

```bash
npm --prefix web run build
cp -R web/dist/. internal/webassets/dist/
go build -o bin/nectar ./cmd/nectar
```

For a local server without a Docker daemon:

```bash
NECTAR_ADDR=127.0.0.1:8080 \
NECTAR_DATA_DIR="$(mktemp -d)" \
NECTAR_INIT_TOKEN=replace-with-a-random-token \
./bin/nectar
```

Canonical checks are exposed through the [Makefile](Makefile). Frontend dependencies are pinned in `web/package.json` and `web/pnpm-lock.yaml`; Go dependencies are pinned by `go.mod` and `go.sum`.

## Architecture and API

The public API is versioned under `/api/v1`; its contract is in [api/openapi.yaml](api/openapi.yaml). Product and architecture decisions are described in [docs/PRODUCT_AND_TECHNICAL_PLAN.md](docs/PRODUCT_AND_TECHNICAL_PLAN.md).

The process embeds the React build, owns SQLite state, talks to the local Manager through the Docker Engine API, and creates Swarm services directly. Traefik reads service labels through its Swarm provider and owns ACME certificate storage in a dedicated volume.

## Security

Read [SECURITY.md](SECURITY.md) before exposing Nectar. Protect the Docker socket, do not expose the bootstrap token, and keep the initial HTTP setup port on a trusted network until the control plane is behind HTTPS.

Report vulnerabilities privately through GitHub's security-advisory interface. Do not open public issues containing credentials, host fingerprints, Docker tokens, private registry details, or production logs with secrets.

## Contributing

Contributions are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md), follow the AngularJS/Conventional Commit format documented in [AGENTS.md](AGENTS.md), and review [ROADMAP.md](ROADMAP.md) before starting a large feature.

## License

Nectar is licensed under the [GNU Affero General Public License v3.0 only](LICENSE), expressed as `AGPL-3.0-only`. If you modify Nectar and let users interact with it over a network, the AGPL requires that those users can obtain the corresponding source.
