#!/bin/bash
# -----------------------------------------------------------------------------
# Kafka Topic Verification Script
# -----------------------------------------------------------------------------
# This script verifies that Terraform-managed topics exist in the Kafka cluster
# with the expected configurations.
#
# Usage: ./verify-topics.sh [environment]
#   environment: Topic prefix (default: dev)
#
# Prerequisites:
#   - Docker with running Kafka containers
#   - jq for JSON parsing (optional, for prettier output)
# -----------------------------------------------------------------------------

set -e

ENV="${1:-dev}"
BROKER_CONTAINER="kafka-1"
KAFKA_BIN="/opt/kafka/bin"
BOOTSTRAP="localhost:9092"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=============================================="
echo "Kafka Topic Verification"
echo "Environment: ${ENV}"
echo "=============================================="
echo ""

# Expected topics
EXPECTED_TOPICS=(
    "${ENV}-orders"
    "${ENV}-order-events"
    "${ENV}-payments"
    "${ENV}-customers"
    "${ENV}-products"
    "${ENV}-dlq-orders"
    "${ENV}-dlq-payments"
)

# Check if Kafka container is running
if ! docker ps | grep -q "${BROKER_CONTAINER}"; then
    echo -e "${RED}Error: Kafka container '${BROKER_CONTAINER}' is not running${NC}"
    echo "Start the cluster with: docker compose up -d"
    exit 1
fi

# Function to check if topic exists
check_topic() {
    local topic=$1
    docker exec ${BROKER_CONTAINER} ${KAFKA_BIN}/kafka-topics.sh \
        --bootstrap-server ${BOOTSTRAP} \
        --list 2>/dev/null | grep -q "^${topic}$"
}

# Function to get topic details
get_topic_details() {
    local topic=$1
    docker exec ${BROKER_CONTAINER} ${KAFKA_BIN}/kafka-topics.sh \
        --bootstrap-server ${BOOTSTRAP} \
        --describe \
        --topic "${topic}" 2>/dev/null
}

# Function to get topic config
get_topic_config() {
    local topic=$1
    docker exec ${BROKER_CONTAINER} ${KAFKA_BIN}/kafka-configs.sh \
        --bootstrap-server ${BOOTSTRAP} \
        --entity-type topics \
        --entity-name "${topic}" \
        --describe 2>/dev/null
}

echo "Checking expected topics..."
echo ""

FOUND=0
MISSING=0

for topic in "${EXPECTED_TOPICS[@]}"; do
    if check_topic "${topic}"; then
        echo -e "${GREEN}✓${NC} ${topic}"
        FOUND=$((FOUND + 1))
    else
        echo -e "${RED}✗${NC} ${topic} (MISSING)"
        MISSING=$((MISSING + 1))
    fi
done

echo ""
echo "----------------------------------------------"
echo "Summary: ${FOUND} found, ${MISSING} missing"
echo "----------------------------------------------"
echo ""

# Show detailed topic information
echo "=============================================="
echo "Topic Details"
echo "=============================================="
echo ""

for topic in "${EXPECTED_TOPICS[@]}"; do
    if check_topic "${topic}"; then
        echo -e "${YELLOW}Topic: ${topic}${NC}"
        echo "----------------------------------------------"
        get_topic_details "${topic}" | grep -v "^$"
        echo ""
        echo "Configuration:"
        get_topic_config "${topic}" | grep -v "^$" | tail -n +2
        echo ""
    fi
done

# Verify specific configurations
echo "=============================================="
echo "Configuration Validation"
echo "=============================================="
echo ""

# Check compacted topics
echo "Checking compacted topics..."
for topic in "${ENV}-customers" "${ENV}-products"; do
    if check_topic "${topic}"; then
        config=$(get_topic_config "${topic}")
        if echo "${config}" | grep -q "cleanup.policy=compact"; then
            echo -e "${GREEN}✓${NC} ${topic} has cleanup.policy=compact"
        else
            echo -e "${RED}✗${NC} ${topic} should have cleanup.policy=compact"
        fi
    fi
done

echo ""
echo "Checking min.insync.replicas..."
for topic in "${EXPECTED_TOPICS[@]}"; do
    if check_topic "${topic}"; then
        config=$(get_topic_config "${topic}")
        if echo "${config}" | grep -q "min.insync.replicas=2"; then
            echo -e "${GREEN}✓${NC} ${topic} has min.insync.replicas=2"
        else
            echo -e "${YELLOW}!${NC} ${topic} min.insync.replicas not explicitly set to 2"
        fi
    fi
done

echo ""
echo "=============================================="
echo "Verification Complete"
echo "=============================================="

if [ ${MISSING} -eq 0 ]; then
    echo -e "${GREEN}All expected topics exist!${NC}"
    exit 0
else
    echo -e "${RED}${MISSING} topics are missing. Run 'terraform apply' to create them.${NC}"
    exit 1
fi
