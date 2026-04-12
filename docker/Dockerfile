############################################
# Dockge Runtime Image
#
# All binaries are pre-built by the CI runner (no compilation happens here):
#   - Go backend binary:        dist/dockge-server
#   - Go healthcheck binary:    dist/healthcheck
#   - Docker CLI (stripped):    dist/docker
#   - Docker Compose (stripped): dist/docker-compose
#   - Frontend static assets:   frontend-dist/
#
# Base image: gcr.io/distroless/base-debian12
#   ✓ glibc  — required by the docker CLI binary
#   ✓ CA certificates
#   ✓ tzdata
#   ✗ NO shell, NO apt, NO package manager, NO curl, NO cron
#
# Our Go binaries are CGO_ENABLED=0 static — they don't need glibc at all,
# but glibc is present anyway for the docker CLI.
############################################
FROM gcr.io/distroless/base-debian12 AS release

# Docker toolchain — stripped binaries supplied by the CI runner.
COPY dist/docker          /usr/local/bin/docker
COPY dist/docker-compose  /usr/local/lib/docker/cli-plugins/docker-compose

# Go backend binary — static, stripped, cross-compiled by the CI runner.
COPY dist/dockge-server  /app/dockge-server

# Pre-built frontend static assets.
COPY frontend-dist/      /app/frontend-dist/

# /app/data is created at first startup by os.MkdirAll in main.go.
# distroless has no shell so we cannot RUN mkdir here.

WORKDIR /app

ENV DOCKGE_IS_CONTAINER=1

VOLUME /app/data
EXPOSE 5001

# The healthcheck subcommand is built into the main binary — no separate
# binary needed. It hits GET /health on the running server.
HEALTHCHECK --interval=60s --timeout=30s --start-period=60s --retries=5 \
    CMD ["/app/dockge-server", "healthcheck"]

# No dumb-init needed: Go handles SIGTERM/SIGINT via signal.Notify +
# graceful http.Server.Shutdown in main.go.
# distroless has no shell so CMD must be exec (JSON array) form.
CMD ["/app/dockge-server"]
