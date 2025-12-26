# Kontrol - Flow Diagrams

## 1. Create Resource Flow

```
┌──────┐
│ User │ POST /api/v1/resources
└───┬──┘
    │
    ▼
┌─────────────────────────────────────────┐
│ API Server                              │
│ - INSERT resources                      │
│ - generation = 1                        │
│ - revision = 1                          │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ PostgreSQL                              │
│ resources table updated                 │
└────────────┬────────────────────────────┘
             │
             ▼ (30s poll)
┌─────────────────────────────────────────┐
│ Reconciler                              │
│ - Find: resources.gen != applied.gen    │
│ - Lock resource_applied_states          │
│ - Apply to K8s with annotations         │
│ - Update applied_states (gen=1, rev=1)  │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Kubernetes                              │
│ Deployment created with:                │
│ - kontrol/resource-id: "123"            │
│ - kontrol/generation: "1"               │
│ - kontrol/revision: "1"                 │
└────────────┬────────────────────────────┘
             │
             ▼ (Watch event)
┌─────────────────────────────────────────┐
│ Watcher                                 │
│ - Lock resource_current_states          │
│ - Read annotations from K8s             │
│ - Update current_states (gen=1, rev=1)  │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ All Synced ✅                           │
│ resources.gen = 1                       │
│ applied_states.gen = 1                  │
│ current_states.gen = 1                  │
└─────────────────────────────────────────┘
```

---

## 2. Update Resource Flow

```
┌──────┐
│ User │ PUT /api/v1/resources/:id
└───┬──┘
    │ {"desired_spec": {"replicas": 5}}
    ▼
┌─────────────────────────────────────────┐
│ API Server                              │
│ - UPDATE resources                      │
│ - generation = 2 (++)                   │
│ - revision = 2 (++)                     │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Status: Out-of-Sync                     │
│ resources.gen = 2                       │
│ applied_states.gen = 1 (stale)          │
└────────────┬────────────────────────────┘
             │
             ▼ (Reconciler detects)
┌─────────────────────────────────────────┐
│ Reconciler                              │
│ - Apply updated spec to K8s             │
│ - Update annotations (gen=2, rev=2)     │
│ - Update applied_states (gen=2, rev=2)  │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Watcher                                 │
│ - Detect K8s change                     │
│ - Update current_states (gen=2, rev=2)  │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ All Synced ✅                           │
│ All generations = 2                     │
└─────────────────────────────────────────┘
```

---

## 3. Rollback Flow

```
┌──────┐
│ User │ PUT /api/v1/resources/:id
└───┬──┘
    │ {"revision": 1}  ← Roll back to rev 1
    ▼
┌─────────────────────────────────────────┐
│ API Server                              │
│ - UPDATE resources                      │
│ - generation = 3 (still increases!)     │
│ - revision = 1 (rolls back)             │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Reconciler                              │
│ - Apply rev 1 spec to K8s               │
│ - Annotations: gen=3, rev=1             │
│ - Update applied_states                 │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Result                                  │
│ generation = 3 (change history)         │
│ revision = 1 (logical rollback)         │
└─────────────────────────────────────────┘
```

---

## 4. Drift Detection Flow

```
┌───────────────┐
│ Manual Change │ kubectl edit deployment nginx
└───────┬───────┘
        │
        ▼
┌─────────────────────────────────────────┐
│ Kubernetes                              │
│ Deployment modified                     │
│ resourceVersion changed                 │
└────────────┬────────────────────────────┘
             │
             ▼ (Watch event)
┌─────────────────────────────────────────┐
│ Watcher                                 │
│ - Detect resourceVersion change         │
│ - Check annotations                     │
│ - If annotations missing → skip         │
│ - If present → update current_states    │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Reconciler (next poll)                  │
│ - Detect: desired != current            │
│ - Reapply desired spec                  │
│ - Restore annotations                   │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Drift Corrected ✅                      │
│ K8s matches desired state               │
└─────────────────────────────────────────┘
```

---

## 5. Error Handling Flow

```
┌──────────────┐
│ Reconciler   │ Apply fails (quota exceeded)
└───────┬──────┘
        │
        ▼
┌─────────────────────────────────────────┐
│ Update resource_applied_states          │
│ - status = "error"                      │
│ - error_message = "quota exceeded"      │
│ - generation NOT updated (stays old)    │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Status: Out-of-Sync (Error)             │
│ resources.gen = 2                       │
│ applied_states.gen = 1 (failed)         │
│ applied_states.status = "error"         │
└────────────┬────────────────────────────┘
             │
             ▼ (30s later)
┌─────────────────────────────────────────┐
│ Reconciler Retry                        │
│ - Detects gen mismatch                  │
│ - Retries apply                         │
└─────────────────────────────────────────┘
```

---

## 6. Delete Flow

```
┌──────┐
│ User │ DELETE /api/v1/resources/:id
└───┬──┘
    │
    ▼
┌─────────────────────────────────────────┐
│ API Server                              │
│ - UPDATE generation++ (mark changed)    │
│ - Soft delete (set deleted_at)          │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Reconciler                              │
│ - Detect deleted resource               │
│ - Delete from K8s                       │
│ - Hard delete from DB (CASCADE)         │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ Cleanup Complete ✅                     │
│ - K8s resource removed                  │
│ - DB records removed (all 3 tables)     │
└─────────────────────────────────────────┘
```
