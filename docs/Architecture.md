# Kontrol - Architecture

## System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        External Clients                      │
│                    (curl, UI, CI/CD)                         │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP REST
                           ▼
┌───────────────────────────────────────────────────────────────┐
│                      API Server                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  POST   /api/v1/resources                               │ │
│  │  GET    /api/v1/resources                               │ │
│  │  GET    /api/v1/resources/:id                           │ │
│  │  PUT    /api/v1/resources/:id                           │ │
│  │  DELETE /api/v1/resources/:id                           │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  - Fiber v3                                                   │
│  - CORS enabled (all domains)                                 │
│  - Port 8080                                                  │
└──────────────────────────┬────────────────────────────────────┘
                           │
                           ▼
┌───────────────────────────────────────────────────────────────┐
│                     PostgreSQL Database                       │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  resources                                              │ │
│  │  - id, cluster_id, namespace, kind, name                │ │
│  │  - desired_spec, generation, revision                   │ │
│  ├─────────────────────────────────────────────────────────┤ │
│  │  resource_current_states                                │ │
│  │  - resource_id, spec, generation, revision              │ │
│  │  - k8s_resource_version                                 │ │
│  ├─────────────────────────────────────────────────────────┤ │
│  │  resource_applied_states                                │ │
│  │  - resource_id, spec, generation, revision              │ │
│  │  - status, error_message                                │ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────┬────────────────────────────────────┘
                           │
                    ┌──────┴──────┐
                    │             │
                    ▼             ▼
        ┌───────────────┐  ┌───────────────┐
        │    Watcher    │  │  Reconciler   │
        │               │  │               │
        │ - Watch API   │  │ - Poll 30s    │
        │ - Update      │  │ - Compare     │
        │   current     │  │   gen/rev     │
        │   states      │  │ - Apply K8s   │
        └───────┬───────┘  └───────┬───────┘
                │                  │
                └──────────┬───────┘
                           │ K8s Client
                           ▼
                ┌──────────────────┐
                │   Kubernetes     │
                │                  │
                │  - Deployments   │
                │  - Services      │
                │  - ConfigMaps    │
                └──────────────────┘
```

## Three Loops

### Loop 1: API Server
```
User request → Validate → Update resources table
                        → generation++
                        → revision++ (or set manually)
```

### Loop 2: Watcher (Real-time)
```
K8s Watch API event
    ↓
Read annotations (kontrol/generation, kontrol/revision)
    ↓
Lock resource_current_states row
    ↓
Compare k8s_resource_version
    ↓
If changed: Update spec, generation, revision
    ↓
Commit
```

### Loop 3: Reconciler (Every 30s)
```
Poll all resources for cluster
    ↓
Find: resources.generation != resource_applied_states.generation
    ↓
Lock resource_applied_states row
    ↓
Apply to K8s with Server-Side Apply
    ↓
Add annotations: kontrol/generation, kontrol/revision
    ↓
Update applied_states: spec, generation, revision, status
    ↓
Commit
```

## Table Ownership

| Table | Writer | Reader | Lock Contention |
|-------|--------|--------|-----------------|
| resources | API Server | Worker | Low (user updates) |
| resource_applied_states | Reconciler | API Server | Medium (30s poll) |
| resource_current_states | Watcher | API Server | High (K8s events) |

**No cross-table locks** → No blocking between components
