# Exercise 02: Topic Creation

## Learning Objectives

- Understand what Kafka topics are
- Create topics with different configurations
- Describe and inspect topics
- Delete topics

## Background

A topic is a category or feed name to which records are published. Topics in Kafka are always multi-subscriber; that is, a topic can have zero, one, or many consumers that subscribe to the data written to it.

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

### Task 1: Create Your First Topic

Create a simple topic called `my-first-topic`:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 --create --topic my-first-topic
```

### Task 2: List All Topics

View all topics in the cluster:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 --list
```

**Question**: What topics do you see? Are there any you didn't create?

### Task 3: Describe a Topic

Get detailed information about your topic:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe \
  --topic my-first-topic
```

**Questions**: 
- How many partitions does the topic have?
- What is the replication factor?

### Task 4: Create a Topic with Custom Configuration

Create a topic with specific partitions and replication factor:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic orders \
  --partitions 3 \
  --replication-factor 1
```

Describe this topic and observe the differences.

### Task 5: Create a Topic with Retention Policy

Create a topic with a 1-hour retention period:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic short-lived-events \
  --config retention.ms=3600000
```

Describe the topic to see the configuration:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe \
  --topic short-lived-events
```

### Task 6: Modify Topic Configuration

Change the retention period of an existing topic:
```bash
./kafka-configs.sh --bootstrap-server localhost:9092 \
  --entity-type topics \
  --entity-name short-lived-events \
  --alter \
  --add-config retention.ms=7200000
```

Verify the change by describing the topic again.

### Task 7: Delete a Topic

Delete the `short-lived-events` topic:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --delete \
  --topic short-lived-events
```

List topics again to verify deletion.

## Verification

You've successfully completed this exercise when you can:
- ✅ Create topics with default settings
- ✅ Create topics with custom partition counts
- ✅ Create topics with custom configurations
- ✅ List and describe topics
- ✅ Delete topics

## Key Concepts

- **Partition**: A topic is split into partitions for parallelism and scalability
- **Replication Factor**: Number of copies of data (limited by number of brokers)
- **Retention**: How long Kafka keeps messages (time or size-based)

## Cleanup

Keep the broker running for the next exercise, or stop it with:
```bash
docker compose down
```

## Next Steps

Continue to [Exercise 03: Producer Basics](../03-producer-basics/)
