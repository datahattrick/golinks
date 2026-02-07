FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
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

# OpenShift runs containers as a random UID in group 0.
# Set group ownership to 0 and grant group read+execute so the
# assigned UID can access all files and directories.
RUN chown -R 0:0 /app && chmod -R g+rX /app

# Declare non-root user.  OpenShift overrides the UID at runtime
# but the image must signal non-root intent for restricted SCCs.
USER 1001

EXPOSE 3000

CMD ["./golinks"]
