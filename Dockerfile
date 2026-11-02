# Stage 1: Build static binary
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk --no-cache add ca-certificates git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary without CGO
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/api .

# Stage 2: Minimal & secure runtime
FROM alpine:3.21

# Install runtime dependencies and setup unprivileged user
RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -u 65532 -g nonroot nonroot

WORKDIR /

# Copy static binary from builder stage
COPY --from=builder --chown=nonroot:nonroot /out/api /api

EXPOSE 8081

USER nonroot

ENTRYPOINT ["/api"]
