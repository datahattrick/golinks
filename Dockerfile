FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o golinks ./cmd/server

# Runtime image
FROM alpine:3.20

WORKDIR /app

# Copy binary and assets
COPY --from=builder /app/golinks .
COPY --from=builder /app/views ./views
COPY --from=builder /app/static ./static

EXPOSE 3000

CMD ["./golinks"]
