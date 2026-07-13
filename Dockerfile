# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with the render tag (enables GROK_ACCOUNTS_JSON bootstrap)
RUN CGO_ENABLED=0 GOOS=linux go build -tags render -trimpath \
    -ldflags="-s -w" -o /out/grok-proxy-cli ./cmd/grok-proxy-cli

# --- Runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/grok-proxy-cli /app/grok-proxy-cli

# Default config — can be overridden by env vars
ENV GROK_DATA_DIR=/data/GrokDesktop
ENV PORT=10000
ENV HOST=0.0.0.0
EXPOSE 10000

# grok-proxy-cli serve --listen 0.0.0.0:$PORT
CMD ["/app/grok-proxy-cli", "serve", "--listen", "0.0.0.0:10000"]
