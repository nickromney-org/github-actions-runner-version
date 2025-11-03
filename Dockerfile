# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -trimpath -ldflags="-w -s" -o github-release-version-checker .

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/github-release-version-checker .

# Create non-root user
RUN adduser -D -u 1000 appuser && \
    chown -R appuser:appuser /app

USER appuser

# Health check - verify binary is executable
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=1 \
    CMD ["/app/github-release-version-checker", "--version"] || exit 1

ENTRYPOINT ["/app/github-release-version-checker"]
