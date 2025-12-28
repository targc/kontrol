# RBAC Setup and Kubeconfig Generation

This guide shows how to create a ServiceAccount with specific RBAC permissions and generate a kubeconfig file for Kontrol.

## When to Use This

Use ServiceAccount-based kubeconfig when:
- Running Kontrol **outside** the Kubernetes cluster
- Want to limit Kontrol's permissions (principle of least privilege)
- Need separate credentials per cluster
- Deploying in multi-cluster environments

**Note:** If running Kontrol **inside** the cluster, you don't need this - just omit `KONTROL_KUBECONFIG` to use in-cluster config.

## Prerequisites

- `kubectl` configured with cluster admin access
- `jq` installed (for kubeconfig generation script)

## Option 1: Cluster-Wide Access

Use this for managing resources across all namespaces.

### Step 1: Create ServiceAccount

```bash
kubectl create serviceaccount kontrol-sa -n default
```

### Step 2: Create ClusterRole

```yaml
# kontrol-clusterrole.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kontrol-role
rules:
# Allow managing common workload resources
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Allow managing core resources
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets", "namespaces"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Allow managing networking resources
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Allow managing batch resources
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

Apply it:
```bash
kubectl apply -f kontrol-clusterrole.yaml
```

### Step 3: Create ClusterRoleBinding

```yaml
# kontrol-clusterrolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kontrol-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kontrol-role
subjects:
- kind: ServiceAccount
  name: kontrol-sa
  namespace: default
```

Apply it:
```bash
kubectl apply -f kontrol-clusterrolebinding.yaml
```

## Option 2: Namespace-Scoped Access

Use this to limit Kontrol to specific namespace(s).

### Step 1: Create ServiceAccount

```bash
kubectl create serviceaccount kontrol-sa -n my-namespace
```

### Step 2: Create Role (instead of ClusterRole)

```yaml
# kontrol-role.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kontrol-role
  namespace: my-namespace
rules:
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

Apply it:
```bash
kubectl apply -f kontrol-role.yaml
```

### Step 3: Create RoleBinding

```yaml
# kontrol-rolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kontrol-binding
  namespace: my-namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kontrol-role
subjects:
- kind: ServiceAccount
  name: kontrol-sa
  namespace: my-namespace
```

Apply it:
```bash
kubectl apply -f kontrol-rolebinding.yaml
```

## Generate Kubeconfig

### Method 1: Automated Script

Create this script (`generate-kubeconfig.sh`):

```bash
#!/bin/bash

# Configuration
SERVICE_ACCOUNT_NAME="kontrol-sa"
NAMESPACE="default"
KUBECONFIG_OUTPUT="kontrol-kubeconfig.yaml"

# Get cluster information
CLUSTER_NAME=$(kubectl config view --minify -o jsonpath='{.clusters[0].name}')
CLUSTER_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
CLUSTER_CA=$(kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')

# Create token secret (required for K8s 1.24+)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: ${SERVICE_ACCOUNT_NAME}-token
  namespace: ${NAMESPACE}
  annotations:
    kubernetes.io/service-account.name: ${SERVICE_ACCOUNT_NAME}
type: kubernetes.io/service-account-token
EOF

# Wait for token to be created
echo "Waiting for token to be created..."
sleep 2

# Get token from secret
TOKEN=$(kubectl get secret ${SERVICE_ACCOUNT_NAME}-token -n ${NAMESPACE} -o jsonpath='{.data.token}' | base64 -d)

# Generate kubeconfig
cat > ${KUBECONFIG_OUTPUT} <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ${CLUSTER_CA}
    server: ${CLUSTER_SERVER}
  name: ${CLUSTER_NAME}
contexts:
- context:
    cluster: ${CLUSTER_NAME}
    user: ${SERVICE_ACCOUNT_NAME}
    namespace: ${NAMESPACE}
  name: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
current-context: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
users:
- name: ${SERVICE_ACCOUNT_NAME}
  user:
    token: ${TOKEN}
EOF

echo "Kubeconfig generated: ${KUBECONFIG_OUTPUT}"
echo ""
echo "Test it with:"
echo "  kubectl --kubeconfig=${KUBECONFIG_OUTPUT} get pods"
```

Make it executable and run:
```bash
chmod +x generate-kubeconfig.sh
./generate-kubeconfig.sh
```

### Method 2: Manual Steps

#### 1. Create Token Secret (K8s 1.24+)

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: kontrol-sa-token
  namespace: default
  annotations:
    kubernetes.io/service-account.name: kontrol-sa
type: kubernetes.io/service-account-token
EOF
```

#### 2. Get Token

```bash
TOKEN=$(kubectl get secret kontrol-sa-token -n default -o jsonpath='{.data.token}' | base64 -d)
echo $TOKEN
```

#### 3. Get Cluster CA Certificate

```bash
CA_CERT=$(kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')
```

#### 4. Get Cluster Server URL

```bash
SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
echo $SERVER
```

#### 5. Create Kubeconfig File

```yaml
# kontrol-kubeconfig.yaml
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: <CA_CERT>
    server: <SERVER>
  name: kontrol-cluster
contexts:
- context:
    cluster: kontrol-cluster
    user: kontrol-sa
    namespace: default
  name: kontrol-context
current-context: kontrol-context
users:
- name: kontrol-sa
  user:
    token: <TOKEN>
```

Replace `<CA_CERT>`, `<SERVER>`, and `<TOKEN>` with the values from steps 2-4.

## Test the Kubeconfig

```bash
# Test connection
kubectl --kubeconfig=kontrol-kubeconfig.yaml get pods

# Test specific permissions
kubectl --kubeconfig=kontrol-kubeconfig.yaml get deployments
kubectl --kubeconfig=kontrol-kubeconfig.yaml create deployment test --image=nginx
kubectl --kubeconfig=kontrol-kubeconfig.yaml delete deployment test
```

## Use with Kontrol

### Environment Variable

```bash
export KONTROL_KUBECONFIG=/path/to/kontrol-kubeconfig.yaml
export KONTROL_CLUSTER_ID=production
go run cmd/worker/main.go
```

### .env File

```bash
KONTROL_DB_URL=postgres://user:pass@localhost:5432/kontrol
KONTROL_KUBECONFIG=/path/to/kontrol-kubeconfig.yaml
KONTROL_CLUSTER_ID=production
```

### Programmatic Usage

```go
package main

import (
    "github.com/targc/kontrol/pkg/worker"
    "github.com/targc/kontrol/pkg/database"
)

func main() {
    db, _ := database.Connect(cfg)

    w, err := worker.NewWorker(
        db,
        "production",
        "/path/to/kontrol-kubeconfig.yaml",
    )

    ctx := context.Background()
    w.Start(ctx)
}
```

## Multi-Cluster Setup

For managing multiple clusters, create separate ServiceAccounts and kubeconfigs:

```bash
# Cluster 1 (production)
./generate-kubeconfig.sh \
  --service-account=kontrol-sa \
  --namespace=default \
  --output=kubeconfig-production.yaml

# Cluster 2 (staging) - switch context first
kubectl config use-context staging
./generate-kubeconfig.sh \
  --service-account=kontrol-sa \
  --namespace=default \
  --output=kubeconfig-staging.yaml
```

Run separate workers:
```bash
# Worker 1
KONTROL_KUBECONFIG=kubeconfig-production.yaml \
KONTROL_CLUSTER_ID=production \
go run cmd/worker/main.go

# Worker 2
KONTROL_KUBECONFIG=kubeconfig-staging.yaml \
KONTROL_CLUSTER_ID=staging \
go run cmd/worker/main.go
```

## Security Best Practices

1. **Least Privilege**: Only grant permissions Kontrol actually needs
2. **Namespace Isolation**: Use namespace-scoped Roles when possible
3. **Token Rotation**: Regularly rotate ServiceAccount tokens
4. **Audit Logging**: Enable K8s audit logging to track Kontrol's actions
5. **Secure Storage**: Store kubeconfig files securely (vault, secrets manager)

## Troubleshooting

### "forbidden: User cannot list resource"

The ServiceAccount lacks necessary RBAC permissions. Check:
```bash
kubectl auth can-i list deployments --as=system:serviceaccount:default:kontrol-sa
```

### Token Expired

ServiceAccount tokens don't expire by default, but if using time-bound tokens, regenerate:
```bash
kubectl delete secret kontrol-sa-token -n default
kubectl apply -f kontrol-sa-token-secret.yaml
```

### Cannot Connect to Cluster

Verify server URL in kubeconfig matches cluster:
```bash
kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}'
```

## Cleanup

Remove all created resources:
```bash
# Delete bindings
kubectl delete clusterrolebinding kontrol-binding
# OR for namespace-scoped
kubectl delete rolebinding kontrol-binding -n my-namespace

# Delete roles
kubectl delete clusterrole kontrol-role
# OR for namespace-scoped
kubectl delete role kontrol-role -n my-namespace

# Delete ServiceAccount and token
kubectl delete serviceaccount kontrol-sa -n default
kubectl delete secret kontrol-sa-token -n default
```
