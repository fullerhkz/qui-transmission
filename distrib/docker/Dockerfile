# Build stage
FROM node:22.18-alpine AS frontend-builder

# Install git for cloning
RUN apk add --no-cache git

# Update and enable Corepack
RUN npm install -g corepack@latest && \
    corepack enable

WORKDIR /app/web

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web ./
RUN pnpm run build

# Go build stage
# Use BUILDPLATFORM to build on native architecture (fast)
FROM --platform=$BUILDPLATFORM golang:1.26-alpine3.22 AS go-builder

# Install build dependencies
RUN apk add --no-cache git

# Cross-compilation arguments from Docker BuildKit
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN mkdir -p internal/transmissionqbt
COPY internal/transmissionqbt/go.mod ./internal/transmissionqbt/go.mod
RUN go mod download

# Copy source code
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=""
ARG BUILD_DATE=""
ARG POLAR_ORG_ID=""

# Set cross-compilation environment variables
ENV GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

# Handle ARM variants
RUN case "${TARGETARCH}" in \
      "arm") \
        case "${TARGETVARIANT}" in \
          "v6") export GOARM=6 ;; \
          "v7") export GOARM=7 ;; \
        esac \
      ;; \
    esac

# Build the application
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/fullerhkz/qui-transmission/internal/buildinfo.Version=${VERSION} -X github.com/fullerhkz/qui-transmission/internal/buildinfo.Commit=${GIT_COMMIT} -X github.com/fullerhkz/qui-transmission/internal/buildinfo.Date=${BUILD_DATE} -X main.PolarOrgID=${POLAR_ORG_ID}" \
    -o qui-transmission ./cmd/qui-transmission

# Final stage
FROM alpine:3.23

LABEL org.opencontainers.image.source="https://github.com/fullerhkz/qui-transmission"

# Set environment variables for config paths
ENV HOME="/config" \
    XDG_CONFIG_HOME="/config" \
    XDG_DATA_HOME="/config"

# Install only what the binary and supported container workflows need at runtime.
# The Go binary is statically linked (CGO_ENABLED=0) so no libc/openssl needed.
# curl and bash stay for the External Programs feature and existing container workflows.
# ca-certificates: TLS connections to Transmission, trackers, update checks
# tzdata: timezone display in logs and UI
RUN apk --no-cache add ca-certificates curl tzdata bash

WORKDIR /config

# Declare volume for persistent data
VOLUME /config

# Copy binary
COPY --from=go-builder /app/qui-transmission /usr/local/bin/

EXPOSE 7476

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7476/health || exit 1

ENTRYPOINT ["/usr/local/bin/qui-transmission"]
CMD ["serve"]
