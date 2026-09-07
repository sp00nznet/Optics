# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Dependencies first so source edits don't re-download the module cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -g '' appuser

COPY --from=builder /app/server .

RUN mkdir -p /data/thumbnails && chown -R appuser:appuser /data

USER appuser

EXPOSE 8080

CMD ["./server"]
