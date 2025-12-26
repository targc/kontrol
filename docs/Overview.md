# Kontrol - Overview

## What is Kontrol?

Kubernetes resource controller using PostgreSQL as source of truth for multi-cluster management.

## Core Concepts

### Three Tables
```
resources              → Desired state (user intent)
resource_current_states → Actual K8s state (from Watch API)
resource_applied_states → Last applied state (from reconciler)
```

### Generation & Revision
- **Generation**: Always increases (1→2→3→4→5) - tracks all changes
- **Revision**: Logical version, can decrease on rollback (1→2→3→2)

### K8s Annotations
```yaml
kontrol/resource-id: "123"
kontrol/generation: "5"
kontrol/revision: "3"
```

## Status Detection

```
Out-of-Sync:    resources.generation != resource_applied_states.generation
Pending Applied: resource_applied_states.generation != resource_current_states.generation
Synced:         All three generations match
```

## Components

```
┌─────────────┐       ┌──────────────┐       ┌─────────────┐
│ API Server  │       │   Worker     │       │ PostgreSQL  │
│             │       │              │       │             │
│ - CRUD API  │◄─────►│ - Watcher    │◄─────►│ 3 Tables    │
│ - Fiber v3  │       │ - Reconciler │       │             │
│ - Port 8080 │       │              │       │             │
└─────────────┘       └──────┬───────┘       └─────────────┘
                             │
                             ▼
                      ┌──────────────┐
                      │  Kubernetes  │
                      └──────────────┘
```

## Flow

```
User → API → Update resources table (generation++)
                    ↓
              Reconciler detects change
                    ↓
              Apply to K8s with annotations
                    ↓
              Update resource_applied_states
                    ↓
              Watcher detects K8s change
                    ↓
              Update resource_current_states
                    ↓
              All synced ✅
```
