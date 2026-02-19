# Faucet for Legible Money devnet/testnet.
#
# Requires legbled:latest to be built first:
#   docker build -f deployment/docker/Dockerfile.cosmos -t legbled .
#
# Build (from repo root):
#   docker build -f tools/faucet/Dockerfile -t lgm-faucet .

# ──── Stage 1: Build faucet binary ────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
COPY tools/agent-sdk/go.mod ./tools/agent-sdk/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /faucet ./tools/faucet

# ──── Stage 2: Runtime (legbled image provides the CLI for signing txs) ────
FROM legbled:latest

COPY --from=builder /faucet /usr/local/bin/faucet

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:8088/health || exit 1

EXPOSE 8088

ENTRYPOINT ["faucet"]
CMD []
