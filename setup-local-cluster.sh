## ============ cluster clean ============

k3d cluster delete kontrol-cluster-local
k3d registry delete registry.localhost

## ============ cluster setup ============

k3d registry create registry.localhost --port 5000

k3d cluster create kontrol-cluster-local \
--image rancher/k3s:v1.31.11-k3s1 \
--servers 1 \
--agents 1 \
--registry-use k3d-registry.localhost:5000 \
-p "15432:5432@loadbalancer" \
-p "16379:6379@loadbalancer"

CURRENT_CONTEXT=$(kubectl config current-context)
if [ "$CURRENT_CONTEXT" != "k3d-kontrol-cluster-local" ]; then
    echo "Error: Current kubectl context is '$CURRENT_CONTEXT'"
    echo "Expected: 'k3d-kontrol-cluster-local'"
    echo "Please switch to the correct cluster context before running this script."
    exit 1
fi
echo "Verified: Running on kontrol-cluster-local cluster"

REGISTRY=k3d-registry.localhost:5000

mkdir -p ./tmp || true

k3d kubeconfig get kontrol-cluster-local > ./tmp/k3d-local-kubeconfig.yaml
