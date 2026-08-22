# syntax=docker/dockerfile:1.7

FROM node:24.13.0-alpine3.23 AS web-build
WORKDIR /src/web
RUN corepack enable && corepack prepare pnpm@11.0.9 --activate
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN npm run build

FROM golang:1.26.5-alpine3.23 AS go-build
ARG VERSION=dev
ARG COMMIT=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=web-build /src/web/dist/ ./internal/webassets/dist/
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X github.com/nectarops/nectar/internal/version.Version=${VERSION} -X github.com/nectarops/nectar/internal/version.Commit=${COMMIT}" \
    -o /out/nectar ./cmd/nectar

FROM alpine:3.23.5
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S nectar && \
    adduser -S -G nectar -h /var/lib/nectar nectar
COPY --from=go-build /out/nectar /usr/local/bin/nectar
RUN mkdir -p /var/lib/nectar && chown nectar:nectar /var/lib/nectar

# Docker socket access is root-equivalent and its group ID varies by host. The
# Swarm deployment runs this image as root only when the socket is mounted.
USER nectar
EXPOSE 8080
VOLUME ["/var/lib/nectar"]
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=6 \
  CMD wget -qO- http://127.0.0.1:8080/health/ready >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/nectar"]
