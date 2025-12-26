# Kontrol - API Specification

## Base URL
```
http://localhost:8080/api/v1
```

## Endpoints

### 1. Create Resource
```
POST /api/v1/resources
```

**Request:**
```json
{
  "cluster_id": "prod",
  "namespace": "default",
  "kind": "Deployment",
  "name": "nginx",
  "api_version": "apps/v1",
  "desired_spec": {
    "apiVersion": "apps/v1",
    "kind": "Deployment",
    "spec": {
      "replicas": 3
    }
  }
}
```

**Response:** `201 Created`
```json
{
  "id": 123,
  "cluster_id": "prod",
  "namespace": "default",
  "kind": "Deployment",
  "name": "nginx",
  "generation": 1,
  "revision": 1
}
```

---

### 2. Get Resource
```
GET /api/v1/resources/:id
```

**Response:** `200 OK`
```json
{
  "id": 123,
  "cluster_id": "prod",
  "namespace": "default",
  "kind": "Deployment",
  "name": "nginx",
  "api_version": "apps/v1",
  "desired_spec": {...},
  "generation": 2,
  "revision": 2,
  "status": "synced"
}
```

**Status Values:**
- `pending`: Not yet applied
- `out-of-sync`: Desired changed, not applied
- `synced`: All generations match

---

### 3. List Resources
```
GET /api/v1/resources?cluster_id=prod
```

**Response:** `200 OK`
```json
{
  "resources": [
    {
      "id": 123,
      "cluster_id": "prod",
      "namespace": "default",
      "kind": "Deployment",
      "name": "nginx",
      "generation": 2,
      "revision": 2
    }
  ],
  "total": 1
}
```

---

### 4. Update Resource
```
PUT /api/v1/resources/:id
```

**Request:**
```json
{
  "desired_spec": {
    "spec": {
      "replicas": 5
    }
  },
  "revision": 3
}
```

**Notes:**
- `desired_spec`: Replaces entire spec
- `revision`: Optional, defaults to `revision + 1`
- `generation`: Auto-incremented

**Response:** `200 OK`
```json
{
  "id": 123,
  "generation": 3,
  "revision": 3,
  "status": "pending"
}
```

---

### 5. Delete Resource
```
DELETE /api/v1/resources/:id
```

**Response:** `202 Accepted`
```json
{
  "id": 123,
  "status": "deleting",
  "message": "Resource marked for deletion"
}
```

**Notes:**
- Soft delete (sets `deleted_at`)
- Increments `generation`
- Reconciler will delete from K8s

---

### 6. Health Check
```
GET /health
```

**Response:** `200 OK`
```json
{
  "status": "healthy"
}
```

---

## Error Responses

```json
{
  "error": "Error message"
}
```

**Status Codes:**
- `400` Bad Request
- `404` Not Found
- `500` Internal Server Error
- `202` Accepted (async operations)
