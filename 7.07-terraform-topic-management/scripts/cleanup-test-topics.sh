#!/bin/bash
# -----------------------------------------------------------------------------
# Kafka Test Topic Cleanup Script
# -----------------------------------------------------------------------------
# This script removes all topics created during Terraform testing.
# It identifies topics by their environment prefix and deletes them.
#
# Usage: ./cleanup-test-topics.sh [prefix]
#   prefix: Topic prefix pattern to match (default: matches test prefixes)
#
# Prerequisites:
#   - Docker with running Kafka containers
#
# WARNING: This script DELETES topics. Use with caution!
# -----------------------------------------------------------------------------

set -e

BROKER_CONTAINER="kafka-1"
KAFKA_BIN="/opt/kafka/bin"
BOOTSTRAP="localhost:9092"

# Test environment prefixes to clean up
TEST_PREFIXES=(
    "test-"
    "integration-"
    "repl-test-"
    "dynamic-"
    "update-"
    "idempotent-"
    "cleanup-"
    "outputs-"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=============================================="
echo "Kafka Test Topic Cleanup"
echo "=============================================="
echo ""

# Check if Kafka container is running
if ! docker ps | grep -q "${BROKER_CONTAINER}"; then
    echo -e "${RED}Error: Kafka container '${BROKER_CONTAINER}' is not running${NC}"
    echo "Start the cluster with: docker compose up -d"
    exit 1
fi

# Get all topics
ALL_TOPICS=$(docker exec ${BROKER_CONTAINER} ${KAFKA_BIN}/kafka-topics.sh \
    --bootstrap-server ${BOOTSTRAP} \
    --list 2>/dev/null)

# Find topics matching test prefixes
TOPICS_TO_DELETE=()

for topic in ${ALL_TOPICS}; do
    for prefix in "${TEST_PREFIXES[@]}"; do
        if [[ "${topic}" == ${prefix}* ]]; then
            TOPICS_TO_DELETE+=("${topic}")
            break
        fi
    done
done

# Handle custom prefix argument
if [ -n "$1" ]; then
    TOPICS_TO_DELETE=()
    for topic in ${ALL_TOPICS}; do
        if [[ "${topic}" == $1* ]]; then
            TOPICS_TO_DELETE+=("${topic}")
        fi
    done
fi

if [ ${#TOPICS_TO_DELETE[@]} -eq 0 ]; then
    echo -e "${GREEN}No test topics found to delete.${NC}"
    exit 0
fi

echo "Found ${#TOPICS_TO_DELETE[@]} test topic(s) to delete:"
echo ""
for topic in "${TOPICS_TO_DELETE[@]}"; do
    echo "  - ${topic}"
done
echo ""

# Confirmation prompt
read -p "Are you sure you want to delete these topics? (yes/no): " confirm
if [[ "${confirm}" != "yes" ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo "Deleting topics..."
echo ""

DELETED=0
FAILED=0

for topic in "${TOPICS_TO_DELETE[@]}"; do
    echo -n "Deleting ${topic}... "
    if docker exec ${BROKER_CONTAINER} ${KAFKA_BIN}/kafka-topics.sh \
        --bootstrap-server ${BOOTSTRAP} \
        --delete \
        --topic "${topic}" 2>/dev/null; then
        echo -e "${GREEN}OK${NC}"
        DELETED=$((DELETED + 1))
    else
        echo -e "${RED}FAILED${NC}"
        FAILED=$((FAILED + 1))
    fi
done

echo ""
echo "----------------------------------------------"
echo "Cleanup Summary"
echo "----------------------------------------------"
echo -e "Deleted: ${GREEN}${DELETED}${NC}"
echo -e "Failed:  ${RED}${FAILED}${NC}"
echo ""

# Verify deletion
echo "Verifying deletion..."
sleep 2

REMAINING=0
for topic in "${TOPICS_TO_DELETE[@]}"; do
    if docker exec ${BROKER_CONTAINER} ${KAFKA_BIN}/kafka-topics.sh \
        --bootstrap-server ${BOOTSTRAP} \
        --list 2>/dev/null | grep -q "^${topic}$"; then
        echo -e "${YELLOW}Warning: ${topic} still exists (may take time to delete)${NC}"
        REMAINING=$((REMAINING + 1))
    fi
done

if [ ${REMAINING} -eq 0 ]; then
    echo -e "${GREEN}All test topics successfully deleted!${NC}"
else
    echo -e "${YELLOW}${REMAINING} topic(s) still pending deletion.${NC}"
    echo "This is normal - Kafka topic deletion is asynchronous."
fi

echo ""
echo "=============================================="
echo "Cleanup Complete"
echo "=============================================="
