# Exercise 07: Batching and Offset Commits

## Learning Objectives

- Understand auto-commit behavior and its implications
- Learn about message batching in consumers
- Implement manual offset commits
- Understand at-least-once vs at-most-once semantics
- Handle commit failures and edge cases

## Background

Offset management is crucial for reliable message processing. Kafka consumers can commit offsets automatically or manually, each with different trade-offs:

- **Auto-commit**: Simple but can lead to message loss or duplicates
- **Manual commit**: More control but requires careful implementation

Understanding when and how offsets are committed is essential for building reliable systems.

This exercise provides implementations in both **Go** (using franz-go) and **Node.js** (using kafkajs).

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Create a topic for this exercise:
```bash
docker exec -it broker bash
cd /opt/kafka/bin
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic orders \
  --partitions 3
```

3. Populate the topic with test data:
```bash
for i in {1..100}; do
  echo "Order-$i"
done | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders

exit  # Exit the broker container
```

## Tasks

### Task 1: Understanding Auto-Commit

Create a simple consumer with auto-commit enabled (the default behavior).

**Go version:**
```bash
cd auto-commit
go run main.go
```

**Observe**:
- Messages are processed one by one with 2-second processing time
- Offsets are committed automatically every 3 seconds
- The consumer simulates a crash at message 8
- Because auto-commit runs periodically, offsets may have already been committed before the crash

**Question**: What happens if the consumer crashes between auto-commits? Run it again and observe that some messages may be reprocessed.

### Task 2: Manual Commit - At Least Once

Now let's implement manual commits to ensure at-least-once delivery.

**Go version:**
```bash
cd manual-commit
go run main.go
```

**Observe**:
- Each message is committed immediately after processing (500ms processing time per message)
- This ensures no message loss but has performance overhead
- If you stop the consumer (Ctrl+C), it will resume from the last committed offset
- This guarantees at-least-once delivery semantics

### Task 3: Batch Commits

Committing after every message is slow. Let's batch commits for better performance.

**Go version:**
```bash
cd batch-commit
go run main.go
```

**Observe**:
- Messages are fetched in batches (up to 5 records per poll)
- All records in a batch are processed together (2 second simulated processing)
- Commit happens after each batch is processed
- Much better performance than per-message commits
- Trade-off: If crash occurs, unprocessed messages in the batch might be reprocessed

Stop with Ctrl+C to observe graceful shutdown behavior.

## Comparison of Strategies

| Strategy | Message Loss Risk | Duplicate Risk | Performance | Use Case |
|----------|------------------|----------------|-------------|----------|
| Auto-commit | High | Low | Best | Non-critical data, idempotent processing |
| Per-message commit | Low | High | Worst | Critical data, small volume |
| Batch commit | Low | Medium | Good | High throughput requirements |

## Best Practices

1. **Use manual commits** for critical data
2. **Batch commits** for better performance with acceptable reprocessing
3. **Always commit on graceful shutdown** to avoid unnecessary reprocessing
4. **Make processing idempotent** when possible to handle duplicates safely
5. **Monitor consumer lag** to ensure commits are happening regularly

## Cleanup

```bash
docker compose down
```

## Next Steps

Continue to [Exercise 08: Retry Mechanism](../08-retry-mechanism/) to learn about error handling and retry strategies with Dead Letter Queues.
