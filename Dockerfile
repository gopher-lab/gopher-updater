# --- Builder ---
ARG GO_VERSION=1.25.1
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# CGO_ENABLED=0 builds a statically linked binary.
RUN CGO_ENABLED=0 go build -o /app/updater ./cmd/updater

# --- Production ---
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/updater /app/updater
ENTRYPOINT ["/app/updater"]
