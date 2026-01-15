# Exercise 05: Topic Partitions

## Learning Objectives

- Understand how partitions enable parallelism and scalability
- See how messages are distributed across partitions
- Control message partitioning with keys
- Understand partition leadership and replicas
- Work with multiple consumers in a consumer group

## Background

Partitions are the unit of parallelism in Kafka. Each partition is an ordered, immutable sequence of records. Partitions allow Kafka to:
- Scale horizontally by adding more partitions
- Process messages in parallel
- Provide fault tolerance through replication

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Connect to the broker:
```bash
docker exec -it broker bash
cd /opt/kafka/bin
```

## Tasks

### Task 1: Create a Multi-Partition Topic

Create a topic with 5 partitions:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic orders \
  --partitions 5
```

Describe the topic to see partition details:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe \
  --topic orders
```

**Observe**: 
- Each partition has a Leader
- The Replicas column (should show 1 since we have one broker)
- The Isr (In-Sync Replicas) column

### Task 2: Observe Round-Robin Partitioning

Produce messages WITHOUT keys (they'll be distributed round-robin):
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic orders
```

Send 10 messages:
```
Order 1
Order 2
Order 3
Order 4
Order 5
Order 6
Order 7
Order 8
Order 9
Order 10
```

Now consume and see which partition each message went to:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --from-beginning \
  --property print.partition=true \
  --property print.offset=true \
  --property print.key=true
```

**Question**: Are messages evenly distributed across partitions?

### Task 3: Use Keys for Partition Assignment

Now produce messages WITH keys. Messages with the same key always go to the same partition:

```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Send messages with customer IDs as keys:
```
customer-123:Order A
customer-456:Order B
customer-123:Order C
customer-789:Order D
customer-123:Order E
customer-456:Order F
customer-789:Order G
customer-123:Order H
```

Consume and observe partitioning:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --from-beginning \
  --property print.partition=true \
  --property print.key=true \
  --property print.value=true \
  --property key.separator=" | "
```

**Question**: Do all messages for `customer-123` go to the same partition?

### Task 4: Understand Partition Ordering

Messages within a partition are ordered, but across partitions they are not. Let's demonstrate:

Create a topic with timestamps:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic time-series \
  --partitions 3
```

Produce timestamped messages:
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic time-series
```

Send messages:
```
2025-12-03T10:00:00 Event A
2025-12-03T10:01:00 Event B
2025-12-03T10:02:00 Event C
2025-12-03T10:03:00 Event D
2025-12-03T10:04:00 Event E
2025-12-03T10:05:00 Event F
```

Consume with partition and offset info:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic time-series \
  --from-beginning \
  --property print.partition=true \
  --property print.offset=true
```

**Observe**: Messages may not appear in chronological order globally, but within each partition they are ordered.

### Task 5: Consumer Group Partition Assignment

Open three terminal windows and connect to the broker in each. Create a consumer group with multiple consumers:

```bash
# Terminal 1
docker exec -it broker bash
cd /opt/kafka/bin
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --group order-processors \
  --from-beginning \
  --consumer-property client.id=consumer-1 \
  --property print.partition=true

# Terminal 2
docker exec -it broker bash
cd /opt/kafka/bin
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --group order-processors \
  --from-beginning \
  --consumer-property client.id=consumer-2 \
  --property print.partition=true

# Terminal 3
docker exec -it broker bash
cd /opt/kafka/bin
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --group order-processors \
  --from-beginning \
  --consumer-property client.id=consumer-3 \
  --property print.partition=true
```

In a fourth terminal, check partition assignment:
```bash
docker exec -it broker bash
cd /opt/kafka/bin
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group order-processors \
  --describe
```

**Observe**: 
- Each consumer (consumer-1, consumer-2, consumer-3) is assigned different partitions
- The partitions are distributed among the 3 consumers
- You can see which client.id is handling which partitions

Now produce some messages and see them distributed among the consumers:
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Send:
```
key1:Message 1
key2:Message 2
key3:Message 3
key4:Message 4
key5:Message 5
```

### Task 6: Increase Partition Count

You can increase (but not decrease) partition count:

```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --alter \
  --topic orders \
  --partitions 8
```

Describe the topic to verify:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe \
  --topic orders
```

**Important**: Existing messages stay in their original partitions. Only new messages use the new partitions.

### Task 7: Partition Reassignment After Scaling

Stop all consumers (`Ctrl+C`) and restart them. Check the partition assignment again:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group order-processors \
  --describe
```

**Observe**: Partitions are rebalanced among the consumers.

### Task 8: Manual Partition Selection

You can produce to a specific partition:

```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --property "parse.partition=true" \
  --property "key.separator=:"
```

Send messages in format `partition:message`:
```
0:This goes to partition 0
3:This goes to partition 3
7:This goes to partition 7
```

Verify:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic orders \
  --partition 0 \
  --from-beginning
```

## Verification

You've successfully completed this exercise when you can:
- ✅ Create topics with multiple partitions
- ✅ Understand round-robin distribution (no key)
- ✅ Use keys to control partition assignment
- ✅ Understand partition ordering guarantees
- ✅ Work with consumer groups and partition assignment
- ✅ Increase partition count
- ✅ Manually specify partitions

## Key Concepts

- **Partition**: Ordered log of messages; unit of parallelism
- **Partition Key**: Determines which partition a message goes to (hash of key % partition count)
- **Ordering**: Guaranteed within a partition, not across partitions
- **Partition Assignment**: How partitions are distributed among consumers in a group
- **Rebalancing**: Redistribution of partitions when consumers join/leave a group

## Best Practices

- Choose partition count based on expected throughput
- Use keys when order matters for related messages
- Partition count should be >= number of consumers for parallelism
- Don't create too many partitions (impacts performance)
- You can increase but not decrease partition count

## Cleanup

Keep the broker running for the next exercise, or stop it with:
```bash
docker compose down
```

## Next Steps

Continue to [Exercise 1.06: Offset Management](../1.06-offset-management/)
