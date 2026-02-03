# Legible Money Node Dockerfile
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm ci

# Copy source
COPY . .

# Build TypeScript
RUN npm run build

# Production image
FROM node:20-alpine

WORKDIR /app

# Install production dependencies only
COPY package*.json ./
RUN npm ci --production

# Copy built files
COPY --from=builder /app/dist ./dist

# Create data directory
RUN mkdir -p /data /secrets

# Default environment
ENV NODE_ENV=production
ENV LOG_LEVEL=info
ENV DATA_DIR=/data
ENV CONFIG_PATH=/config/node.json
ENV GENESIS_PATH=/config/genesis.json

# Expose ports
# P2P
EXPOSE 26656
# RPC
EXPOSE 8545
# WebSocket
EXPOSE 8546
# Prometheus
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8545/health || exit 1

# Run node
CMD ["node", "dist/cmd/node.js"]
