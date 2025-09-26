# js8d Docker Image
# Multi-stage build for minimal production image

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    alsa-lib-dev

# Set working directory
WORKDIR /src

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    alsa-lib \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1001 -S js8d && \
    adduser -u 1001 -S js8d -G js8d

# Create required directories
RUN mkdir -p /etc/js8d /var/lib/js8d /var/log/js8d /run/js8d && \
    chown -R js8d:js8d /etc/js8d /var/lib/js8d /var/log/js8d /run/js8d

# Copy binary from build stage
COPY --from=builder /src/js8d /usr/local/bin/js8d
RUN chmod +x /usr/local/bin/js8d

# Copy configuration and web assets
COPY --from=builder /src/configs/config.example.yaml /etc/js8d/config.yaml
COPY --from=builder /src/web /usr/share/js8d/web
COPY --from=builder /src/docs /usr/share/js8d/docs

# Set permissions
RUN chown -R js8d:js8d /etc/js8d /usr/share/js8d

# Switch to non-root user
USER js8d

# Set working directory
WORKDIR /var/lib/js8d

# Environment variables
ENV JS8D_CONFIG_DIR=/etc/js8d
ENV JS8D_DATA_DIR=/var/lib/js8d
ENV JS8D_LOG_DIR=/var/log/js8d
ENV JS8D_RUN_DIR=/run/js8d

# Expose web interface port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Default command
CMD ["/usr/local/bin/js8d", "-config", "/etc/js8d/config.yaml"]

# Labels
LABEL org.opencontainers.image.title="js8d"
LABEL org.opencontainers.image.description="Headless JS8Call daemon with web interface"
LABEL org.opencontainers.image.vendor="js8d project"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/dougsko/js8d"
LABEL org.opencontainers.image.documentation="https://github.com/dougsko/js8d/tree/main/docs"