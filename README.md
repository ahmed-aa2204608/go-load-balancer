# Go Load Balancer

A Go-based load balancer running in front of three backend API servers, all backed by a shared PostgreSQL database and orchestrated with Docker Compose.

## Current Status

The stack is working end to end:

- PostgreSQL starts inside Compose and is reachable by the backend containers over the Docker network.
- Three backend servers expose `GET /health` and `POST /createuser`.
- The load balancer listens on port `8000` and proxies traffic to `server1`, `server2`, and `server3`.
- The test script now prints request analytics including throughput, status code counts, and latency percentiles.

## Architecture

```text
Client
  |
  v
Load Balancer (:8000)
  |----> server1 (:3030)
  |----> server2 (:3030)
  |----> server3 (:3030)
              |
              v
       PostgreSQL (:5432 in container, :5435 on host)
```

## Services

- `loadbalancer` in [`src/`](/mnt/c/Users/kingh/Documents/Side-Projects/New%20folder%20(3)/src) exposes port `8000`
- `server1`, `server2`, `server3` in [`server/`](/mnt/c/Users/kingh/Documents/Side-Projects/New%20folder%20(3)/server) expose port `3030` internally and are published as `3030`, `3031`, and `3032` on the host
- `postgres` exposes `5432` internally and `5435` on the host

## Load Balancing Algorithms

The load balancer code currently supports:

- round robin
- least connections via `lc`
- least response time via `lrt`

The application reads the environment variable `LB_ALGORITHM` in [`src/main.go`](/mnt/c/Users/kingh/Documents/Side-Projects/New%20folder%20(3)/src/main.go). If it is unset, the balancer falls back to round robin.

If you want to switch algorithms in Docker Compose, set `LB_ALGORITHM` on the `loadbalancer` service. The Go code does not read `LB_ALGO`.

Example values:

```bash
LB_ALGORITHM=lc
LB_ALGORITHM=lrt
```

## Running the Stack

From the project root:

```bash
docker compose up --build
```

That starts:

- PostgreSQL
- three backend API servers
- the load balancer

Useful host ports:

- `http://localhost:8000` for the load balancer
- `http://localhost:3030` for `server1`
- `http://localhost:3031` for `server2`
- `http://localhost:3032` for `server3`
- `localhost:5435` for PostgreSQL access from the host

## API Endpoints

All public requests are expected to go through the load balancer on port `8000`.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | health check endpoint |
| `POST` | `/createuser` | inserts a user into PostgreSQL |

Examples:

```bash
curl http://localhost:8000/health

curl -X POST http://localhost:8000/createuser \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

You can also hit the backend containers directly on `3030`, `3031`, and `3032` for per-instance checks.

## Testing

The project includes [`test.sh`](/mnt/c/Users/kingh/Documents/Side-Projects/New%20folder%20(3)/test.sh), a simple shell-based load script for exercising the load balancer.

Usage:

```bash
bash test.sh health
bash test.sh health 200
bash test.sh createuser 50
```

The script now reports:

- total requests
- success rate
- HTTP error count
- curl error count
- throughput
- status code breakdown
- latency summary: min, p50, p90, p95, p99, max, average
- average connect time
- average TTFB
- average response size
- slowest request number

## Backend Notes

- The backend service builds its Postgres DSN from environment variables in [`server/server.go`](/mnt/c/Users/kingh/Documents/Side-Projects/New%20folder%20(3)/server/server.go).
- In Docker Compose, the backend containers connect to Postgres using the service hostname `postgres` on port `5432`.
- For host-based runs outside Compose, the backend still falls back to `localhost:5435`.
