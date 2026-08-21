# Dock-Weaver

Dock-Weaver is a free, open-source, self-hosted Docker Swarm management panel for small and medium-sized teams running a modest number of servers. It is installed on the first Manager node and provides a Web interface for managing additional Linux servers. Dock-Weaver connects to those servers over SSH, installs a consistent Docker version, joins them to the Swarm, and uses Traefik to provide domain routing and automatic HTTPS for deployed services.

The backend is implemented in Go. The Web frontend uses React, TypeScript, Vite, and shadcn/ui. Dock-Weaver does not require an official cloud account or service, and it does not charge based on the number of servers, nodes, users, or applications.

## Planned Installation Experience

Dock-Weaver will be installed on the first Linux host with a single command. The final download URL will be published with the first release:

```bash
curl -fsSL https://github.com/<owner>/dock-weaver/releases/latest/download/install.sh | sudo bash
```

An explicit Docker Engine version can be selected without editing the script:

```bash
curl -fsSL https://github.com/<owner>/dock-weaver/releases/latest/download/install.sh \
  | sudo bash -s -- --docker-version <version> --advertise-addr <manager-ip>
```

The installer will check the host, install or validate Docker, initialize Docker Swarm when necessary, start Dock-Weaver as a Swarm service, wait for its health check, and print the setup URL and a one-time initialization token. A downloadable script and checksum-verified installation path will also be documented for operators who prefer to inspect the script before running it as root.

This repository is currently in the planning stage. See the complete product and technical plan for details:

- [Product and Technical Plan](docs/PRODUCT_AND_TECHNICAL_PLAN.md)

## License

Dock-Weaver is licensed under the [GNU Affero General Public License v3.0 only](LICENSE), identified by the SPDX expression `AGPL-3.0-only`.

Individuals and organizations may use, study, modify, and distribute this project, including for commercial purposes, subject to the obligations of the license.
