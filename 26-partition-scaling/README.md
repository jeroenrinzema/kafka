# Exercise 26: Partition Scaling

## Learning Objectives

- Understand why Kafka cannot reduce partition counts directly
- Learn the Create-Migrate-Delete pattern for reducing partitions
- Practice migrating data between topics with different partition counts
- Understand the impact on key distribution and consumer groups
- Learn production strategies for zero-downtime partition changes

## Background

### Increasing Partitions

Kafka allows you to **increase** the number of partitions for a topic:

```bash
kafka-topics.sh --alter --topic my-topic --partitions 10 --bootstrap-server localhost:9092
```

However, this has implications:
- Existing messages stay in their original partitions
- New messages with existing keys may go to different partitions
- Key-based ordering guarantees are broken for existing keys

### Decreasing Partitions

Kafka does **NOT** support decreasing the number of partitions. Once a topic has N partitions, you cannot reduce that number directly.

**Why not?**

1. **Data Distribution**: Messages are assigned to partitions based on `hash(key) % numPartitions`. Removing partitions would invalidate this mapping.
2. **Consumer Groups**: Consumers are assigned to specific partitions. Removing partitions would require complex rebalancing and offset translation.
3. **Ordering Guarantees**: Messages are ordered within a partition. Merging partitions would break ordering guarantees per key.
4. **Offset Management**: Consumer offsets are per-partition. Removed partitions would have orphaned offsets.

### The Solution: Create-Migrate-Delete Pattern

To effectively "reduce" partitions:

1. Create a new topic with the desired (smaller) number of partitions
2. Migrate data from the old topic to the new one
3. Switch producers and consumers to the new topic
4. Delete the old topic

## Prerequisites

- Docker and Docker Compose installed
- Completed exercises 1-5 (Kafka fundamentals)

## Setup

Start a 3-broker Kafka cluster:

```bash
docker compose up -d
```

Wait about 15 seconds for the cluster to initialize.

Verify the cluster is ready:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server kafka-1:19092
```

## Tasks

### Task 1: Create an Over-Partitioned Topic

Imagine you initially created a topic with too many partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic user-events \
  --bootstrap-server kafka-1:19092 \
  --partitions 12 \
  --replication-factor 2
```

Verify the partition count:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic user-events \
  --bootstrap-server kafka-1:19092
```

### Task 2: Produce Test Data with Keys

Produce messages with keys (user IDs) to ensure proper distribution:

```bash
for i in {1..100}; do
  echo "user-$((i % 10)):$(printf '{"userId": "user-%d", "event": "click", "timestamp": "%s", "seq": %d}' $((i % 10)) "$(date -u +%Y-%m-%dT%H:%M:%SZ)" $i)"
done | docker exec -i kafka-1 /opt/kafka/bin/kafka-console-producer.sh \
  --topic user-events \
  --bootstrap-server kafka-1:19092 \
  --property parse.key=true \
  --property key.separator=:
```

This creates 100 messages distributed across 10 different user keys.

### Task 3: Verify Data Distribution

Check how messages are distributed across partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic user-events \
  --bootstrap-server kafka-1:19092 \
  --from-beginning \
  --property print.key=true \
  --property print.partition=true \
  --timeout-ms 10000 2>/dev/null | head -20
```

Notice that messages with the same key go to the same partition.

### Task 4: Attempt to Reduce Partitions (This Will Fail)

Try to reduce partitions directly:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --alter \
  --topic user-events \
  --partitions 3 \
  --bootstrap-server kafka-1:19092
```

You'll see an error: `The topic user-events currently has 12 partition(s); 3 would not be an increase.`

This confirms that Kafka does not support reducing partitions.

### Task 5: Create the Target Topic

Create a new topic with the desired number of partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic user-events-v2 \
  --bootstrap-server kafka-1:19092 \
  --partitions 3 \
  --replication-factor 2
```

Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic user-events-v2 \
  --bootstrap-server kafka-1:19092
```

### Task 6: Migrate Data

Migrate all existing data from the old topic to the new one, preserving keys:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic user-events \
  --bootstrap-server kafka-1:19092 \
  --from-beginning \
  --property print.key=true \
  --property key.separator=: \
  --timeout-ms 10000 2>/dev/null | \
docker exec -i kafka-1 /opt/kafka/bin/kafka-console-producer.sh \
  --topic user-events-v2 \
  --bootstrap-server kafka-1:19092 \
  --property parse.key=true \
  --property key.separator=:
```

### Task 7: Verify Migration

Count messages in both topics:

```bash
echo "=== Original topic (12 partitions) ==="
docker exec kafka-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic user-events \
  --bootstrap-server kafka-1:19092 \
  --from-beginning \
  --timeout-ms 5000 2>/dev/null | wc -l

echo ""
echo "=== New topic (3 partitions) ==="
docker exec kafka-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic user-events-v2 \
  --bootstrap-server kafka-1:19092 \
  --from-beginning \
  --timeout-ms 5000 2>/dev/null | wc -l
```

Both should show 100 messages.

### Task 8: Verify Key Distribution in New Topic

Check that messages with the same key still go to the same partition:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic user-events-v2 \
  --bootstrap-server kafka-1:19092 \
  --from-beginning \
  --property print.key=true \
  --property print.partition=true \
  --timeout-ms 10000 2>/dev/null | sort | head -20
```

Messages with the same key should be in the same partition (though the partition number will be different from the original topic).

### Task 9: Delete the Original Topic

Once you've verified the migration and switched all producers/consumers:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --delete \
  --topic user-events \
  --bootstrap-server kafka-1:19092
```

Verify it's gone:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server kafka-1:19092
```

### Task 10: Rename the New Topic (Optional)

If you need to keep the original topic name, create it again and migrate:

```bash
# Recreate with original name but fewer partitions
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic user-events \
  --bootstrap-server kafka-1:19092 \
  --partitions 3 \
  --replication-factor 2

# Migrate from v2 to the recreated topic
docker exec kafka-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic user-events-v2 \
  --bootstrap-server kafka-1:19092 \
  --from-beginning \
  --property print.key=true \
  --property key.separator=: \
  --timeout-ms 10000 2>/dev/null | \
docker exec -i kafka-1 /opt/kafka/bin/kafka-console-producer.sh \
  --topic user-events \
  --bootstrap-server kafka-1:19092 \
  --property parse.key=true \
  --property key.separator=:

# Delete the intermediate topic
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --delete \
  --topic user-events-v2 \
  --bootstrap-server kafka-1:19092
```

Verify the final result:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic user-events \
  --bootstrap-server kafka-1:19092
```

---

## Production Migration Strategies

The console consumer/producer approach works for small datasets, but production systems need more robust solutions.

### Strategy 1: Dual-Write Pattern

For live systems with continuous traffic:

```
                    ┌─────────────────────┐
                    │     Producers       │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Write to BOTH     │
                    └──────────┬──────────┘
                         │           │
              ┌──────────▼───┐ ┌─────▼──────────┐
              │ Old Topic    │ │ New Topic      │
              │ (12 parts)   │ │ (3 parts)      │
              └──────────────┘ └────────────────┘
```

**Steps:**
1. Deploy producer changes to write to BOTH topics
2. Backfill new topic with historical data from old topic
3. Switch consumers to read from new topic
4. Remove dual-write, write only to new topic
5. Delete old topic

### Strategy 2: MirrorMaker 2

Use MirrorMaker 2 for continuous replication:

```bash
# MirrorMaker 2 can replicate to a topic with different partition count
# The target topic's partition count is independent of the source
```

Benefits:
- Continuous replication
- Handles ongoing traffic
- Built-in offset translation

### Strategy 3: Kafka Streams Application

Write a simple Kafka Streams application:

```java
Properties props = new Properties();
props.put(StreamsConfig.APPLICATION_ID_CONFIG, "partition-migration");
props.put(StreamsConfig.BOOTSTRAP_SERVERS_CONFIG, "localhost:9092");

StreamsBuilder builder = new StreamsBuilder();
builder.stream("user-events")
       .to("user-events-v2");

KafkaStreams streams = new KafkaStreams(builder.build(), props);
streams.start();
```

Benefits:
- Exactly-once semantics
- Continuous migration
- Easy to monitor

### Strategy 4: Kafka Connect

Use a connector pair:

1. **Source Connector**: Read from old topic
2. **Sink Connector**: Write to new topic

This provides fault tolerance and monitoring through Kafka Connect.

---

## Key Concepts

### Impact on Key Distribution

When you reduce partitions, keys are rehashed:

| Keys | 12 Partitions | 3 Partitions |
|------|---------------|--------------|
| key-0 | hash(key-0) % 12 = 4 | hash(key-0) % 3 = 1 |
| key-1 | hash(key-1) % 12 = 7 | hash(key-1) % 3 = 1 |
| key-2 | hash(key-2) % 12 = 2 | hash(key-2) % 3 = 2 |

**Important**: Messages with the same key still go to the same partition in the new topic. The partition NUMBER changes, but the guarantee of "same key = same partition" is preserved.

### Consumer Group Considerations

| Aspect | Impact |
|--------|--------|
| **Offsets** | Old consumer group offsets don't apply to new topic |
| **Parallelism** | Fewer partitions = fewer parallel consumers possible |
| **Rebalancing** | Consumers must be configured for new topic |
| **Lag** | New topic starts fresh; no historical lag |

### When to Reduce Partitions

| Reason | Benefit |
|--------|---------|
| Over-provisioned initially | Reduce overhead |
| Traffic decreased | Match capacity to demand |
| Cost optimization | Less replication traffic, less storage |
| Simplified operations | Fewer partitions to monitor |
| Consumer group simplification | Easier assignment |

### When NOT to Reduce Partitions

| Reason | Risk |
|--------|------|
| High throughput topic | May become bottleneck |
| Many parallel consumers | Limits parallelism |
| Expecting growth | Will need to increase again |
| Complex key distribution | May cause hot partitions |

---

## Troubleshooting

### Migration Takes Too Long

For large topics, the console consumer approach is slow. Use:
- Kafka Streams with multiple instances
- MirrorMaker 2 with multiple tasks
- Parallel consumers writing to new topic

### Duplicate Messages After Migration

If consumers process both old and new topics during transition:
- Use idempotent consumers
- Track processed message IDs
- Use exactly-once semantics with Kafka Streams

### Key Distribution Uneven After Migration

With fewer partitions, more keys share each partition:
- Monitor partition sizes
- Consider partition count carefully
- Use custom partitioner if needed

### Consumer Offsets Lost

New topic has no offset history:
- Consumers start from `earliest` or `latest`
- No way to resume from previous position
- Plan consumer restart strategy

---

## Cleanup

Stop all services:

```bash
docker compose down -v
```

---

## Best Practices

1. **Plan partition count carefully** from the start
2. **Test migration** in non-production first
3. **Use dual-write** for zero-downtime migrations
4. **Monitor both topics** during transition
5. **Verify message counts** before deleting old topic
6. **Communicate with consumers** about the switch
7. **Keep old topic briefly** as backup after migration
8. **Document the migration** for future reference

---

## Additional Resources

- [Kafka Topic Operations](https://kafka.apache.org/documentation/#topicconfigs)
- [Kafka Streams Documentation](https://kafka.apache.org/documentation/streams/)
- [MirrorMaker 2 Guide](https://kafka.apache.org/documentation/#georeplication)

## Next Steps

Try these challenges:

1. Write a Kafka Streams application for continuous migration
2. Implement the dual-write pattern with a test producer
3. Use MirrorMaker 2 to migrate between topics
4. Create a monitoring dashboard for migration progress
5. Practice point-in-time migration (only messages before a timestamp)