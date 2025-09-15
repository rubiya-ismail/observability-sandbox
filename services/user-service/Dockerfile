# Stage 1: Builder
FROM golang:1.24.3 AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y git

# Set working directory
WORKDIR /app

# Copy go workspace files first (for proper module resolution)
COPY go.work go.work.sum ./

# Copy shared module
COPY shared/ shared/

# Copy service modules
COPY services/user-service/ services/user-service/
COPY services/order-service/ services/order-service/
COPY services/api-gateway/ services/api-gateway/

# Set working directory to user-service
WORKDIR /app/services/user-service

# Download dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service main.go
RUN echo "✅ Build completed" && ls -lh /app


# Stage 2: Runtime
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app/

# Copy binary from builder stage
COPY --from=builder /app/services/user-service/user-service .

# Change ownership to non-root user
RUN chown appuser:appuser user-service

# Switch to non-root user
USER appuser

# Expose port (adjust based on your service configuration)
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

# Run the binary
CMD ["./user-service"]  

