# Mantis Gateway

API gateway for Mantis Exchange, a cryptocurrency exchange platform.

## Overview

mantis-gateway is the HTTP/WebSocket entry point for all client traffic. It handles:

- REST API endpoints for order management, market data, and account operations
- WebSocket connections for real-time market data streaming
- JWT-based authentication
- Per-IP token bucket rate limiting
- Request routing to backend services (matching engine via gRPC - not yet implemented)

## Project Structure

```
cmd/gateway/main.go        - Application entry point
internal/handler/           - HTTP route handlers (order, market, account)
internal/middleware/         - Middleware (auth, rate limiting)
internal/ws/                - WebSocket hub for real-time data push
internal/config/            - Configuration loaded from environment variables
```

## Configuration

Set the following environment variables:

| Variable              | Default          | Description                        |
|-----------------------|------------------|------------------------------------|
| `PORT`                | `8080`           | HTTP server listen port            |
| `MATCHING_ENGINE_ADDR`| `localhost:9090` | gRPC address of matching engine    |
| `JWT_SECRET`          | `changeme`       | Secret key for JWT validation      |

## How to Run

```bash
# Build
go build -o mantis-gateway ./cmd/gateway

# Run with defaults
./mantis-gateway

# Run with custom config
PORT=3000 JWT_SECRET=mysecret MATCHING_ENGINE_ADDR=engine:9090 ./mantis-gateway
```

## Development

```bash
# Download dependencies
go mod tidy

# Build all packages
go build ./...

# Run directly
go run ./cmd/gateway
```

## API Endpoints

### REST (all under /api/v1)

- `POST   /api/v1/orders`          - Place a new order (requires auth)
- `DELETE /api/v1/orders/:id`      - Cancel an order (requires auth)
- `GET    /api/v1/depth/:symbol`   - Order book depth
- `GET    /api/v1/trades/:symbol`  - Recent trades
- `GET    /api/v1/account`         - Account info (requires auth)
- `GET    /api/v1/account/balances`- Account balances (requires auth)

### WebSocket

- `GET /ws` - Real-time market data stream (subscribe to channels via JSON messages)
