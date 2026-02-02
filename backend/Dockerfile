# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod file and download dependencies
COPY go.mod ./
RUN go mod download || true

# Copy source code
COPY . .

# Tidy and download dependencies
RUN go mod tidy && go mod download

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
