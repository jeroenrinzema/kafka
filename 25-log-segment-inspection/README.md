# Exercise 25: Log Segment Inspection & Forensics

## Learning Objectives

- Understand Kafka's log storage format
- Use `kafka-dump-log.sh` to inspect segment files
- Analyze index files and understand log structure
- Verify compaction results
- Investigate tombstones and delete markers
- Perform forensic analysis on Kafka data

## Background

Understanding how Kafka stores data on disk is essential for:

- **Troubleshooting**: Investigating missing or corrupted messages
- **Capacity planning**: Understanding storage requirements
- **Compaction verification**: Ensuring log compaction works correctly
- **Forensics**: Recovering data or investigating incidents
- **Performance tuning**: Optimizing segment and index configuration

### Kafka Log Structure

```
/tmp/kafka-logs/
└── my-topic-0/                          # Topic partition directory
    ├── 00000000000000000000.log         # Segment file (messages)
    ├── 00000000000000000000.index       # Offset index
    ├── 00000000000000000000.timeindex   # Timestamp index
    ├── 00000000000000000156.log         # Next segment (starts at offset 156)
    ├── 00000000000000000156.index
    ├── 00000000000000000156.timeindex
    ├── 00000000000000000156.snapshot    # Producer state snapshot
    └── leader-epoch-checkpoint          # Leader epoch data
```

> **Note**: The default log directory in the Apache Kafka Docker image is `/tmp/kafka-logs/`. In production, this is typically configured to a persistent volume like `/var/kafka-logs/`.

### Key Concepts

- **Segment**: A chunk of the log, split by size or time
- **Offset Index**: Maps message offsets to file positions
- **Time Index**: Maps timestamps to offsets
- **Active Segment**: The current segment being written to
- **Log Compaction**: Keeps only the latest value per key

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                    Kafka Log Storage                            │
│                                                                 │
│  Topic: orders, Partition: 0                                   │
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐ │
│  │ Segment 0        │  │ Segment 156      │  │ Segment 312  │ │
│  │ (offsets 0-155)  │  │ (offsets 156-311)│  │ (active)     │ │
│  │                  │  │                  │  │              │ │
│  │ .log   (data)    │  │ .log   (data)    │  │ .log  (data) │ │
│  │ .index (offsets) │  │ .index (offsets) │  │ .index       │ │
│  │ .timeindex       │  │ .timeindex       │  │ .timeindex   │ │
│  │                  │  │                  │  │              │ │
│  │ [IMMUTABLE]      │  │ [IMMUTABLE]      │  │ [MUTABLE]    │ │
│  └──────────────────┘  └──────────────────┘  └──────────────┘ │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Completed exercises 1-6 (Kafka fundamentals)
- Completed exercise 14 (Compacted topics) - recommended
- Docker and Docker Compose installed

## Setup

Start the Kafka cluster:

```bash
docker compose up -d
```

Wait about 15 seconds for the cluster to initialize.

Verify the cluster is running:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092
```

## Tasks

### Task 1: Create Topics with Different Configurations

Create a regular topic with small segments (for testing):

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic events \
  --bootstrap-server localhost:9092 \
  --partitions 1 \
  --replication-factor 1 \
  --config segment.bytes=1048576 \
  --config segment.ms=60000
```

Create a compacted topic:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic users \
  --bootstrap-server localhost:9092 \
  --partitions 1 \
  --replication-factor 1 \
  --config cleanup.policy=compact \
  --config segment.bytes=1048576 \
  --config min.cleanable.dirty.ratio=0.1 \
  --config delete.retention.ms=100
```

### Task 2: Produce Test Messages

Produce messages to the events topic:

```bash
docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic events \
  --bootstrap-server localhost:9092 \
  --property "parse.key=true" \
  --property "key.separator=:" << 'EOF'
event-1:{"type": "click", "page": "/home", "timestamp": "2024-01-15T10:00:00Z"}
event-2:{"type": "view", "page": "/products", "timestamp": "2024-01-15T10:00:05Z"}
event-3:{"type": "click", "page": "/products/123", "timestamp": "2024-01-15T10:00:10Z"}
event-4:{"type": "purchase", "product": "123", "amount": 99.99}
event-5:{"type": "view", "page": "/checkout", "timestamp": "2024-01-15T10:00:20Z"}
EOF
```

Produce messages with the same keys to the users topic (for compaction testing):

```bash
docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic users \
  --bootstrap-server localhost:9092 \
  --property "parse.key=true" \
  --property "key.separator=:" << 'EOF'
user-1:{"name": "Alice", "email": "alice@example.com", "status": "active"}
user-2:{"name": "Bob", "email": "bob@example.com", "status": "active"}
user-1:{"name": "Alice Smith", "email": "alice@example.com", "status": "active"}
user-3:{"name": "Charlie", "email": "charlie@example.com", "status": "active"}
user-2:{"name": "Bob", "email": "bob@example.com", "status": "inactive"}
user-1:{"name": "Alice Smith", "email": "alice.smith@example.com", "status": "active"}
EOF
```

### Task 3: Find the Log Directory

Locate the log files on disk:

```bash
docker exec kafka ls -la /tmp/kafka-logs/
```

List the partition directories:

```bash
docker exec kafka ls -la /tmp/kafka-logs/events-0/
docker exec kafka ls -la /tmp/kafka-logs/users-0/
```

You'll see the segment files (.log), index files (.index, .timeindex), and other metadata.

### Task 4: Inspect Log Segment Contents

Use `kafka-dump-log.sh` to view the segment contents:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --print-data-log
```

This shows:
- **baseOffset**: First offset in the batch
- **lastOffset**: Last offset in the batch
- **count**: Number of messages in the batch
- **baseSequence**: Producer sequence number (for idempotence)
- **partitionLeaderEpoch**: Leader epoch when written
- **payload**: The actual message content

### Task 5: Decode Message Details

Get detailed message information including headers:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --print-data-log \
  --deep-iteration
```

The `--deep-iteration` flag expands compressed batches and shows individual records.

### Task 6: Inspect the Offset Index

View the offset index file:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.index
```

This shows the mapping:
- **offset**: Logical message offset
- **position**: Byte position in the .log file

The index is sparse - it doesn't contain every offset, just periodic entries.

### Task 7: Inspect the Time Index

View the timestamp index:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.timeindex
```

This maps:
- **timestamp**: Message timestamp
- **offset**: The offset of messages at that time

Useful for time-based seeks (e.g., "give me messages from yesterday").

### Task 8: Generate More Data to Create Multiple Segments

Produce enough data to trigger segment rolling:

```bash
docker exec kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 5000 \
  --record-size 1000 \
  --throughput 1000 \
  --producer-props bootstrap.servers=localhost:9092
```

Check that multiple segments were created:

```bash
docker exec kafka ls -la /tmp/kafka-logs/events-0/
```

You should see multiple .log files with different starting offsets.

### Task 9: Compare Segment Files

List segments with their sizes:

```bash
docker exec kafka bash -c "ls -lh /tmp/kafka-logs/events-0/*.log"
```

Inspect a newer segment:

```bash
# Find the second segment file name
SEGMENT=$(docker exec kafka bash -c "ls /tmp/kafka-logs/events-0/*.log | tail -1")
echo "Inspecting: $SEGMENT"

docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files $SEGMENT \
  --print-data-log | head -30
```

### Task 10: Verify Offset Continuity

Check that offsets are continuous across segments:

```bash
# Get the last offset of first segment
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --print-data-log | tail -5

# Get the first offset of second segment
SECOND_SEGMENT=$(docker exec kafka bash -c "ls /tmp/kafka-logs/events-0/*.log | sed -n '2p'")
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files $SECOND_SEGMENT \
  --print-data-log | head -5
```

The first offset of segment 2 should immediately follow the last offset of segment 1.

### Task 11: Inspect Compacted Topic Before Compaction

View the users topic log before compaction runs:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/users-0/00000000000000000000.log \
  --print-data-log
```

You should see all 6 messages, including the older versions of user-1 and user-2.

### Task 12: Trigger Log Compaction

Force log compaction to run:

```bash
# Produce more data to make the segment eligible for compaction
for i in {1..100}; do
  echo "user-$((i % 5)):{'name': 'User $i', 'iteration': $i}" | \
    docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
      --topic users \
      --bootstrap-server localhost:9092 \
      --property "parse.key=true" \
      --property "key.separator=:"
done

# Wait a moment for compaction
sleep 10
```

Force a segment roll to make the current segment eligible for compaction:

```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --entity-type topics \
  --entity-name users \
  --add-config segment.ms=1000
```

Wait for compaction:

```bash
sleep 15
```

### Task 13: Verify Compaction Results

After compaction, inspect the log again:

```bash
docker exec kafka ls -la /tmp/kafka-logs/users-0/
```

You may see:
- `.cleaned` files (during compaction)
- `.deleted` files (being removed)
- Fewer or smaller segments

Inspect the compacted data:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/users-0/00000000000000000000.log \
  --print-data-log
```

For each key, only the latest value should remain.

### Task 14: Create and Inspect Tombstones

Tombstones are messages with null values that mark a key for deletion during compaction.

Produce a tombstone:

```bash
# The empty value after the colon creates a tombstone
echo "user-1:" | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic users \
  --bootstrap-server localhost:9092 \
  --property "parse.key=true" \
  --property "key.separator=:" \
  --property "null.marker="
```

Alternatively, use the producer with null value:

```bash
docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic users \
  --bootstrap-server localhost:9092 \
  --property "parse.key=true" \
  --property "key.separator=:" << 'EOF'
user-2:
EOF
```

Inspect the log to see the tombstone:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/users-0/00000000000000000000.log \
  --print-data-log | grep -A5 "user-2"
```

Tombstones have `payload: null` or empty payload.

### Task 15: Check Producer State Snapshots

Producer state snapshots track idempotent producer sequences:

```bash
docker exec kafka ls -la /tmp/kafka-logs/events-0/*.snapshot 2>/dev/null || echo "No snapshots yet"
```

If snapshots exist, they contain producer ID to sequence number mappings.

### Task 16: Inspect Leader Epoch Checkpoint

View the leader epoch data:

```bash
docker exec kafka cat /tmp/kafka-logs/events-0/leader-epoch-checkpoint
```

This file tracks:
- Leader epoch numbers
- Start offset for each epoch
- Used for replica synchronization and leader failover

### Task 17: Verify Index Consistency

Check for index corruption:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.index \
  --verify-index-only
```

A healthy index will show no errors. Corrupted indexes will report mismatches.

### Task 18: Analyze Batch Compression

If compression is enabled, inspect compression ratios:

```bash
docker exec kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 1000 \
  --record-size 1000 \
  --throughput 500 \
  --producer-props bootstrap.servers=localhost:9092 compression.type=gzip
```

Inspect the compressed batches:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --print-data-log | grep -i "compresscodec"
```

### Task 19: Calculate Storage Efficiency

Compare logical data size to actual disk usage:

```bash
# Logical size (from Kafka's perspective)
docker exec kafka /opt/kafka/bin/kafka-log-dirs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic-list events

# Actual disk usage
docker exec kafka du -sh /tmp/kafka-logs/events-0/
```

### Task 20: Forensic Analysis - Find Specific Messages

Search for a specific message by offset:

```bash
# Find message at offset 3
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --print-data-log \
  --deep-iteration | grep -A10 "offset: 3"
```

Search by key pattern:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --print-data-log | grep -B5 -A5 "event-3"
```

### Task 21: Export Log Data for Analysis

Export log contents to a file for offline analysis:

```bash
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --print-data-log > events-dump.txt

head -50 events-dump.txt
```

### Task 22: Understand Segment Configuration

View current segment configuration:

```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type topics \
  --entity-name events
```

Key segment settings:
- `segment.bytes`: Max segment size
- `segment.ms`: Max segment age
- `segment.index.bytes`: Max index size

### Task 23: Monitor Active Segment

Identify the active (current) segment:

```bash
# The active segment is the one currently being written to
# It's usually the last one (highest offset) in the directory
docker exec kafka bash -c "ls -la /tmp/kafka-logs/events-0/*.log | tail -1"
```

The active segment:
- Is being appended to
- Has an open file handle
- Will be rolled when it hits size/time limits

### Task 24: Recovery Log Position

Check the recovery checkpoint:

```bash
docker exec kafka cat /tmp/kafka-logs/recovery-point-offset-checkpoint
```

This file records where recovery should start after a crash.

### Task 25: Log Start Offset Checkpoint

View the log start offset (after retention/compaction):

```bash
docker exec kafka cat /tmp/kafka-logs/log-start-offset-checkpoint
```

This tracks the earliest available offset per partition.

## Key Concepts

### Segment File Format

Each .log file contains batches of records:

```
┌─────────────────────────────────────────────────────────┐
│ Record Batch                                             │
│  ├─ Base Offset                                          │
│  ├─ Batch Length                                         │
│  ├─ Partition Leader Epoch                               │
│  ├─ Magic Byte (version)                                 │
│  ├─ CRC                                                  │
│  ├─ Attributes (compression, timestamp type, etc.)       │
│  ├─ Last Offset Delta                                    │
│  ├─ First Timestamp                                      │
│  ├─ Max Timestamp                                        │
│  ├─ Producer ID                                          │
│  ├─ Producer Epoch                                       │
│  ├─ Base Sequence                                        │
│  └─ Records[]                                            │
│       ├─ Length                                          │
│       ├─ Attributes                                      │
│       ├─ Timestamp Delta                                 │
│       ├─ Offset Delta                                    │
│       ├─ Key                                             │
│       ├─ Value                                           │
│       └─ Headers[]                                       │
└─────────────────────────────────────────────────────────┘
```

### Index Types

| Index | Purpose | Contents |
|-------|---------|----------|
| `.index` | Offset lookup | offset → file position |
| `.timeindex` | Time-based seek | timestamp → offset |
| `.txnindex` | Transaction lookup | (for transactional producers) |

### Compaction States

1. **Dirty**: Segments with potential duplicate keys
2. **Cleanable**: Dirty ratio exceeds threshold
3. **Cleaning**: Being compacted (`.cleaned` files)
4. **Clean**: Only latest values per key

## Troubleshooting

### Log File Corrupted

```bash
# Verify the log integrity
docker exec kafka /opt/kafka/bin/kafka-dump-log.sh \
  --files /tmp/kafka-logs/events-0/00000000000000000000.log \
  --verify-index-only

# If corrupted, Kafka will rebuild indexes on startup
# You can force this by deleting .index and .timeindex files
```

### Missing Messages

```bash
# Check log start offset (messages before this are deleted)
docker exec kafka /opt/kafka/bin/kafka-run-class.sh \
  kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic events \
  --time -2  # -2 = earliest

# Check high watermark
docker exec kafka /opt/kafka/bin/kafka-run-class.sh \
  kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic events \
  --time -1  # -1 = latest
```

### Compaction Not Working

```bash
# Check cleaner status
docker exec kafka cat /tmp/kafka-logs/cleaner-offset-checkpoint

# Verify topic is configured for compaction
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic users \
  --bootstrap-server localhost:9092
```

## Cleanup

Stop all services:

```bash
docker compose down -v
```

Remove any exported files:

```bash
rm -f events-dump.txt
```

## Best Practices

1. **Regular monitoring**: Track segment counts and sizes
2. **Appropriate segment size**: Balance between too many small files and too few large ones
3. **Index monitoring**: Ensure indexes aren't corrupted
4. **Compaction verification**: Periodically verify compaction is working
5. **Backup before investigation**: Copy files before modifying
6. **Use read-only tools**: `kafka-dump-log.sh` is safe; don't modify files directly
7. **Document findings**: Keep records of forensic investigations
8. **Automate health checks**: Script periodic log verification

## Additional Resources

- [Kafka Log Format](https://kafka.apache.org/documentation/#messageformat)
- [Log Compaction](https://kafka.apache.org/documentation/#compaction)
- [Kafka Internals](https://developer.confluent.io/courses/architecture/log/)
- [KIP-186: Log Message Timestamp](https://cwiki.apache.org/confluence/display/KAFKA/KIP-186%3A+Increase+offsets+retention+default+to+7+days)

## Next Steps

Try these challenges:

1. Create a script that validates all log segments in a cluster
2. Build a tool that extracts messages matching a pattern from raw logs
3. Analyze compaction efficiency for a high-churn topic
4. Investigate how transactions appear in log segments
5. Compare storage efficiency with different compression codecs