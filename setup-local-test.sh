#!/bin/bash
set -e

# Configuration
DB_URL="${DB_URL:-postgres://postgres:password@localhost:5432/dev?sslmode=disable}"
CLUSTER_ID="${CLUSTER_ID:-default}"
API_KEY="${API_KEY:-sk_local_test_key_12345}"

# Install Python dependencies
pip install -q bcrypt psycopg2-binary

# Hash and insert using Python
python3 <<PYEOF
import bcrypt
import psycopg2
import uuid

db_url = "$DB_URL"
cluster_id = "$CLUSTER_ID"
api_key = "$API_KEY"

# Hash API key
key_hash = bcrypt.hashpw(api_key.encode(), bcrypt.gensalt(rounds=10)).decode()

# Connect and insert
conn = psycopg2.connect(db_url)
cur = conn.cursor()

# Create cluster
cur.execute("""
    INSERT INTO k_clusters (id, created_at, updated_at)
    VALUES (%s, NOW(), NOW())
    ON CONFLICT (id) DO NOTHING
""", (cluster_id,))

# Create API key
cur.execute("""
    INSERT INTO k_cluster_api_keys (id, cluster_id, key_hash, name, created_at, updated_at)
    VALUES (%s, %s, %s, 'local-dev-key', NOW(), NOW())
""", (str(uuid.uuid4()), cluster_id, key_hash))

conn.commit()
cur.close()
conn.close()
PYEOF

echo ""
echo "=== Kontrol Local Test Data Setup Complete ==="
echo ""
echo "Cluster ID: $CLUSTER_ID"
echo "API Key: $API_KEY"
echo ""
echo "export KONTROL_API_URL=http://localhost:8080"
echo "export KONTROL_API_KEY=$API_KEY"
echo "export KONTROL_CLUSTER_ID=$CLUSTER_ID"
