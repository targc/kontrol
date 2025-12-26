# Kontrol

Kubernetes resource controller using PostgreSQL as source of truth for multi-cluster management.

## Quick Start

### Prerequisites
- Go 1.25.4+
- PostgreSQL 16+
- Kubernetes cluster (with kubeconfig)

### Setup

1. **Clone and install dependencies**
```bash
git clone https://github.com/targc/kontrol.git
cd kontrol
go mod download
```

2. **Configure environment**
```bash
cp .env.example .env
# Edit .env with your settings
```

3. **Start PostgreSQL**
```bash
docker run -d \
  --name kontrol-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=kontrol \
  -p 5432:5432 \
  postgres:16-alpine
```

4. **Run API Server**
```bash
go run cmd/api-server/main.go
```

5. **Run Worker (separate terminal)**
```bash
export CLUSTER_ID=prod
go run cmd/worker/main.go
```

### Usage

**Create a Deployment:**
```bash
curl -X POST http://localhost:8080/api/v1/resources \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": "prod",
    "namespace": "default",
    "kind": "Deployment",
    "name": "nginx",
    "api_version": "apps/v1",
    "desired_spec": {
      "apiVersion": "apps/v1",
      "kind": "Deployment",
      "metadata": {"name": "nginx"},
      "spec": {"replicas": 3}
    }
  }'
```

**Get Resource:**
```bash
curl http://localhost:8080/api/v1/resources/1
```

**Update Resource:**
```bash
curl -X PUT http://localhost:8080/api/v1/resources/1 \
  -H "Content-Type: application/json" \
  -d '{"desired_spec": {"spec": {"replicas": 5}}}'
```

## Documentation

- [Overview](docs/Overview.md) - Core concepts
- [Architecture](docs/Architecture.md) - System design
- [Database Schema](docs/Database-Schema.md) - Tables and queries
- [API Specification](docs/API-Specification.md) - REST API reference
- [Flow Diagrams](docs/Flow.md) - Data flows

## Architecture

```
User → API Server → PostgreSQL ← Worker → Kubernetes
              (source of truth)
```

**Components:**
- **API Server**: REST API (Fiber v3, port 8080)
- **Worker**: Watcher (K8s Watch API) + Reconciler (30s poll)

**3 Tables:**
- `resources`: Desired state
- `resource_current_states`: Actual K8s state
- `resource_applied_states`: Last applied state

## License

MIT
