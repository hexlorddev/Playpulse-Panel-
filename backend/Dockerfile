# Multi-stage build for Go backend
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install required packages
RUN apk --no-cache add \
    ca-certificates \
    curl \
    openjdk11-jre \
    bash \
    zip \
    unzip \
    wget \
    && addgroup -g 1001 -S playpulse \
    && adduser -u 1001 -S playpulse -G playpulse

# Set timezone
RUN apk add --no-cache tzdata
ENV TZ=UTC

# Create necessary directories
RUN mkdir -p /opt/minecraft-servers \
    && mkdir -p /app/backups \
    && mkdir -p /app/uploads \
    && mkdir -p /app/logs \
    && chown -R playpulse:playpulse /opt/minecraft-servers \
    && chown -R playpulse:playpulse /app

# Copy binary from builder stage
COPY --from=builder /app/main /app/playpulse-panel

# Copy configuration
COPY --from=builder /app/.env.example /app/.env

# Set permissions
RUN chmod +x /app/playpulse-panel

# Create non-root user
USER playpulse

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Start the application
CMD ["./playpulse-panel"]