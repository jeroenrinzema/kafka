# Exercise 08: Retry Mechanisms

## Learning Objectives

- Understand error handling patterns in Kafka consumers
- Implement retry mechanisms with exponential backoff
- Learn about dead letter queues (DLQ) for failed messages
- Handle transient vs permanent errors
- Implement circuit breaker patterns
- Understand at-least-once delivery with retries

## Background

In real-world applications, message processing can fail for various reasons:
- **Transient errors**: Network timeouts, temporary service unavailability
- **Permanent errors**: Invalid data format, business logic violations
- **Resource errors**: Database connections, rate limits

A robust retry mechanism helps ensure message processing reliability while preventing infinite retry loops for messages that will never succeed.

## Retry Strategies

1. **Simple Retry**: Retry immediately a fixed number of times
2. **Exponential Backoff**: Increase delay between retries exponentially
3. **Dead Letter Queue (DLQ)**: Move permanently failed messages to a separate topic for investigation

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Create the necessary topics:
```bash
docker exec -it broker bash
cd /opt/kafka/bin

# Main topic for orders
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic orders \
  --partitions 3

# Dead letter queue for permanently failed messages
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic orders-dlq \
  --partitions 1

exit
```

## Tasks

### Task 1: Simple Retry Mechanism

The simple retry approach retries failed messages immediately up to a maximum number of attempts.

```bash
cd simple-retry
go run main.go
```

In another terminal, produce test messages (some will fail):
```bash
docker exec -it broker bash
cd /opt/kafka/bin

# These will succeed
echo "order-1" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
echo "order-2" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders

# This will trigger an error (contains "fail")
echo "order-fail-3" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders

# More successful messages
echo "order-4" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
```

**Observe**:
- Successful messages are processed normally
- Messages containing "fail" trigger errors
- Failed messages are retried up to 3 times
- After max retries, processing continues (message is lost)
- Offset is only committed after successful processing

**Question**: What happens to messages that fail all retry attempts?

### Task 2: Exponential Backoff

Exponential backoff increases the delay between retries, which is useful for transient errors that may resolve over time.

```bash
cd ../exponential-backoff
go run main.go
```

Produce test messages:
```bash
docker exec -it broker bash
cd /opt/kafka/bin

echo "order-1" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
echo "order-transient-2" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
echo "order-3" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
```

**Observe**:
- Messages containing "transient" trigger temporary errors
- Retry delays: 1s, 2s, 4s, 8s (exponential)
- Transient errors eventually succeed after several retries
- Good for handling temporary service unavailability

**Question**: When would exponential backoff be preferred over simple retry?

### Task 3: Dead Letter Queue (DLQ)

Instead of dropping failed messages, send them to a DLQ for later investigation.

```bash
cd ../dead-letter-queue
go run main.go
```

Produce test messages:
```bash
docker exec -it broker bash
cd /opt/kafka/bin

echo "order-1" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
echo "order-permanent-2" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
echo "order-3" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
echo "order-transient-4" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
echo "order-permanent-5" | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic orders
```

**Observe**:
- Transient errors are retried with exponential backoff
- Permanent errors (containing "permanent") are sent to DLQ immediately (no retries)
- Failed messages are preserved in the DLQ topic with error metadata
- Original offsets are committed after DLQ delivery
- No message loss - you can investigate and reprocess DLQ messages later

**Question**: How could you create a separate consumer to monitor and reprocess messages from the DLQ?

## Comparison of Retry Strategies

| Strategy | Pros | Cons | Use Case |
|----------|------|------|----------|
| Simple Retry | Simple implementation | Blocks partition processing | Quick, infrequent errors |
| Exponential Backoff | Handles transient errors well | Still blocks partition | Network/service timeouts |
| Dead Letter Queue | No message loss, preserves failures | Requires DLQ monitoring | Critical data, investigation needed |

## Best Practices

1. **Distinguish error types**: Treat transient and permanent errors differently
2. **Set max retries**: Prevent infinite loops
3. **Use exponential backoff**: For transient errors
4. **Implement DLQ**: For permanent failures
5. **Add metadata**: Store retry count, error messages in headers
6. **Monitor DLQ**: Set up alerts for DLQ accumulation
7. **Make processing idempotent**: Handle duplicate processing from retries
8. **Circuit breaker**: Stop processing when error rate is too high
9. **Graceful degradation**: Continue processing other messages

## Error Handling Decision Tree

```
Message Processing Failed
    ↓
Is it a transient error?
    ├─ Yes → Retry with exponential backoff
    │         ↓
    │    Max retries reached?
    │         ├─ Yes → Send to DLQ
    │         └─ No → Retry
    │
    └─ No (Permanent error)
          ↓
     Send to DLQ immediately
```

## Verification

You've successfully completed this exercise when you can:
- ✅ Implement simple retry logic
- ✅ Use exponential backoff for transient errors
- ✅ Send failed messages to a DLQ
- ✅ Understand when to use each retry strategy
- ✅ Distinguish between transient and permanent errors

## Advanced Topics

- **Retry Topic Pattern**: Use a separate topic for retries to avoid blocking main processing
- **Circuit Breaker Pattern**: Temporarily stop processing when error rate exceeds threshold
- **Retry Headers**: Store retry count and timestamps in message headers
- **Custom Retry Delays**: Implement custom backoff strategies
- **Metrics**: Track retry rates, DLQ size, error types
- **DLQ Monitoring**: Create separate consumers to monitor and reprocess DLQ messages

## Cleanup

```bash
docker compose down
```

## Next Steps

Continue to [Exercise 09: Kaf Introduction](../09-kaf-introduction/) to learn about a modern CLI tool for Kafka.
