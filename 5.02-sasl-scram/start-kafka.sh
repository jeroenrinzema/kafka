#!/bin/bash
set -e

# SCRAM credentials to bootstrap
ADMIN_USER="admin"
ADMIN_PASSWORD="admin-secret"
CLUSTER_ID="${CLUSTER_ID:-4L6g3nShT-eMCtK--X86sw}"
LOG_DIR="${KAFKA_LOG_DIRS:-/tmp/kraft-combined-logs}"

echo "=== Kafka SCRAM Bootstrap Script ==="

# Check if storage is already formatted
if [ ! -f "$LOG_DIR/meta.properties" ]; then
    echo "Formatting storage with SCRAM credentials..."

    # Format storage with SCRAM credentials for admin user
    /opt/kafka/bin/kafka-storage.sh format \
        --cluster-id "$CLUSTER_ID" \
        --config /opt/kafka/config/server.properties \
        --standalone \
        --add-scram "SCRAM-SHA-256=[name=${ADMIN_USER},password=${ADMIN_PASSWORD}]" \
        --add-scram "SCRAM-SHA-512=[name=${ADMIN_USER},password=${ADMIN_PASSWORD}]" \
        --ignore-formatted

    echo "Storage formatted with SCRAM credentials for user: ${ADMIN_USER}"
else
    echo "Storage already formatted, skipping SCRAM bootstrap"
fi

echo "Starting Kafka..."
exec /etc/kafka/docker/run
