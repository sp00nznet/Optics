# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Set GOPRIVATE to prevent Go from trying to fetch internal packages
ENV GOPRIVATE=github.com/optics/*
ENV GOFLAGS="-mod=mod"

# Copy all source code
COPY . .

# Download external dependencies and generate go.sum
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /app/server .

# Create directories
RUN mkdir -p /data/thumbnails && chown -R appuser:appuser /data

USER appuser

EXPOSE 8080

CMD ["./server"]
