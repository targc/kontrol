# Kontrol - Database Schema

## Tables

### 1. resources

**Purpose**: User's desired state (source of truth)

```sql
CREATE TABLE resources (
    id              SERIAL PRIMARY KEY,
    cluster_id      VARCHAR(100) NOT NULL,
    namespace       VARCHAR(255) NOT NULL,
    kind            VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    api_version     VARCHAR(100),

    desired_spec    JSONB NOT NULL,

    generation      INTEGER DEFAULT 1 NOT NULL,
    revision        INTEGER DEFAULT 1 NOT NULL,

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE INDEX idx_resources_cluster_id ON resources(cluster_id);
CREATE INDEX idx_resources_deleted_at ON resources(deleted_at);
```

| Field | Type | Description |
|-------|------|-------------|
| id | SERIAL | Primary key |
| cluster_id | VARCHAR | Target cluster |
| namespace | VARCHAR | K8s namespace |
| kind | VARCHAR | Resource type (Deployment, Service, etc) |
| name | VARCHAR | Resource name |
| api_version | VARCHAR | K8s API version (apps/v1, v1, etc) |
| desired_spec | JSONB | User's desired state |
| generation | INTEGER | Always increases on change |
| revision | INTEGER | Logical version (can decrease) |

---

### 2. resource_current_states

**Purpose**: Actual K8s state (synced by Watcher)

```sql
CREATE TABLE resource_current_states (
    id                      SERIAL PRIMARY KEY,
    resource_id             INTEGER NOT NULL UNIQUE REFERENCES resources(id) ON DELETE CASCADE,

    spec                    JSONB,
    generation              INTEGER,
    revision                INTEGER,
    k8s_resource_version    VARCHAR(100),

    created_at              TIMESTAMP DEFAULT NOW(),
    updated_at              TIMESTAMP DEFAULT NOW(),
    deleted_at              TIMESTAMP
);

CREATE INDEX idx_rcs_resource_id ON resource_current_states(resource_id);
```

| Field | Type | Description |
|-------|------|-------------|
| id | SERIAL | Primary key (same as resource_id) |
| resource_id | INTEGER | FK to resources.id |
| spec | JSONB | Actual K8s spec |
| generation | INTEGER | From kontrol/generation annotation |
| revision | INTEGER | From kontrol/revision annotation |
| k8s_resource_version | VARCHAR | K8s resourceVersion (change detection) |

---

### 3. resource_applied_states

**Purpose**: Last applied state (by Reconciler)

```sql
CREATE TABLE resource_applied_states (
    id              SERIAL PRIMARY KEY,
    resource_id     INTEGER NOT NULL UNIQUE REFERENCES resources(id) ON DELETE CASCADE,

    spec            JSONB,
    generation      INTEGER,
    revision        INTEGER,

    status          VARCHAR(50),
    error_message   TEXT,

    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    deleted_at      TIMESTAMP
);

CREATE INDEX idx_ras_resource_id ON resource_applied_states(resource_id);
CREATE INDEX idx_ras_status ON resource_applied_states(status);
```

| Field | Type | Description |
|-------|------|-------------|
| id | SERIAL | Primary key (same as resource_id) |
| resource_id | INTEGER | FK to resources.id |
| spec | JSONB | Spec that was applied |
| generation | INTEGER | Generation that was applied |
| revision | INTEGER | Revision that was applied |
| status | VARCHAR | success / error |
| error_message | TEXT | Error if apply failed |

---

## Relationships

```
resources (1)
    ↓
    ├─→ resource_current_states (1) [ON DELETE CASCADE]
    └─→ resource_applied_states (1) [ON DELETE CASCADE]
```

---

## Status Detection Queries

### Out-of-Sync Resources
```sql
SELECT r.id, r.kind, r.namespace, r.name
FROM resources r
LEFT JOIN resource_applied_states ras ON r.id = ras.resource_id
WHERE r.cluster_id = 'prod'
  AND r.generation != COALESCE(ras.generation, 0);
```

### Pending Applied
```sql
SELECT r.id, r.kind, r.namespace, r.name
FROM resources r
LEFT JOIN resource_applied_states ras ON r.id = ras.resource_id
LEFT JOIN resource_current_states rcs ON r.id = rcs.resource_id
WHERE ras.generation != COALESCE(rcs.generation, 0);
```

### Synced Resources
```sql
SELECT r.id, r.kind, r.namespace, r.name
FROM resources r
LEFT JOIN resource_applied_states ras ON r.id = ras.resource_id
LEFT JOIN resource_current_states rcs ON r.id = rcs.resource_id
WHERE r.generation = ras.generation
  AND ras.generation = rcs.generation;
```
