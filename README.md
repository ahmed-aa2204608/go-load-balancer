# Go Load Balancer

A simple round-robin load balancer written in Go, sitting in front of 3 backend HTTP servers backed by PostgreSQL.

## Architecture

```
Client → Load Balancer (:8000) → server1 (:3030)
                               → server2 (:3030)
                               → server3 (:3030)
                                    ↓
                               PostgreSQL (:5432)
```

- **Load Balancer** (`src/`) — distributes incoming requests across the 3 servers using round-robin
- **Backend Servers** (`server/`) — expose `/health` and `/createuser` endpoints, connected to a shared Postgres database

## Prerequisites

- [Docker](https://www.docker.com/products/docker-desktop) with Compose

## Getting Started

From the project root:

```bash
docker compose up --build
```

This starts Postgres, builds and runs all 3 backend servers, and starts the load balancer.

## Endpoints

All requests go through the load balancer on port `8000`:

| Method | Path          | Description         |
|--------|---------------|---------------------|
| GET    | `/health`     | Health check        |
| POST   | `/createuser` | Create a new user   |

**Example:**

```bash
curl http://localhost:8000/health

curl -X POST http://localhost:8000/createuser \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com"}'
```
