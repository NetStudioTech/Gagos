# Stage 1: Builder
FROM golang:1.21-bookworm AS builder

WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Download dependencies (will also generate go.sum)
RUN go mod download || go mod tidy

# Copy source code
COPY . .

# Ensure dependencies are correct
RUN go mod tidy

# Build static binary
ARG VERSION=dev
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /gagos ./cmd/gagos

# Stage 2: Minimal runtime
# Using alpine for small size but with CA certs for HTTPS
FROM alpine:3.19

# Install CA certificates and useful shell tools
RUN apk --no-cache add ca-certificates tzdata \
    busybox-extras \
    curl \
    bind-tools \
    iputils \
    net-tools \
    jq \
    bash

# Create non-root user for running the app
RUN addgroup -g 1000 gagos && \
    adduser -u 1000 -G gagos -D -h /home/gagos -s /bin/sh gagos

# Create app directory with restricted permissions
RUN mkdir -p /app/web && chown -R root:root /app && chmod -R 755 /app

# Copy binary (owned by root, executable by all)
COPY --from=builder /gagos /usr/local/bin/gagos

# Copy web assets (read-only for the app)
COPY web/ /app/web/
RUN chmod -R 644 /app/web/* && find /app/web -type d -exec chmod 755 {} \;

# Create writable tmp for terminal sessions
RUN mkdir -p /tmp && chmod 1777 /tmp

# Create data directory for BBolt database (will be mounted as emptyDir in k8s)
RUN mkdir -p /data && chown gagos:gagos /data

WORKDIR /home/gagos

USER gagos

EXPOSE 8080

# Simple healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["gagos"]
