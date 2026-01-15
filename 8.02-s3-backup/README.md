# Exercise 13: Topic Backup to S3 Storage

## Learning Objectives

- Understand Kafka Connect and sink connectors
- Configure S3 Sink Connector for topic archival
- Set up MinIO as S3-compatible object storage
- Implement time-based and size-based file rotation
- Restore topics from S3 backups

## What You'll Build

1. **Kafka Cluster**: Single broker with test topics
2. **MinIO**: S3-compatible object storage for backups
3. **Kafka Connect**: S3 Sink Connector for automated archival
4. **Recovery Tools**: Scripts to restore data from backups

## Prerequisites

- Docker and Docker Compose installed
- Basic understanding of Kafka topics

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Data Flow                                │
│                                                                  │
│  ┌─────────┐     ┌──────────────┐     ┌─────────────────────┐  │
│  │ Kafka   │────▶│ Kafka        │────▶│ MinIO (S3)          │  │
│  │ Topics  │     │ Connect      │     │                     │  │
│  │         │     │ S3 Sink      │     │ topics/orders/...   │  │
│  │ orders  │     │              │     │ topics/events/...   │  │
│  │ events  │     │              │     │                     │  │
│  └─────────┘     └──────────────┘     └─────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Setup

Start all services:

```bash
docker compose up -d
```

Wait about 45 seconds for Kafka Connect to download and install the S3 connector plugin.

Verify all services are running:

```bash
docker compose ps
```

## Tasks

### Task 1: Verify Services

Check Kafka is ready:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server localhost:9092
```

Check Kafka Connect is ready:

```bash
curl -s http://localhost:8083/ | head -1
```

Access MinIO Console at http://localhost:9001
- Username: `minioadmin`
- Password: `minioadmin`

### Task 2: Create Topics

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic orders \
  --partitions 3 --replication-factor 1 \
  --bootstrap-server localhost:9092

docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic events \
  --partitions 2 --replication-factor 1 \
  --bootstrap-server localhost:9092
```

### Task 3: Understand the Connector Configuration

View the S3 sink connector configuration:

```bash
cat s3-sink-connector.json
```

Key settings:

| Setting | Value | Description |
|---------|-------|-------------|
| `topics` | orders,events | Topics to backup |
| `s3.bucket.name` | kafka-backup | Target S3 bucket |
| `flush.size` | 5 | Records per file (for testing) |
| `rotate.interval.ms` | 60000 | Create new file every 60 seconds |
| `format.class` | JsonFormat | Output as JSON |
| `s3.compression.type` | gzip | Compress files |

### Task 4: Deploy the S3 Sink Connector

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d @s3-sink-connector.json
```

Check connector status:

```bash
curl -s http://localhost:8083/connectors/s3-sink/status | jq .
```

You should see `"state": "RUNNING"` for both the connector and its tasks.

### Task 5: Produce Test Messages

Produce messages to the orders topic:

```bash
for i in {1..10}; do
  echo "{\"orderId\": \"order-$i\", \"amount\": $((RANDOM % 100 + 1)), \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
done | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic orders --bootstrap-server localhost:9092
```

Produce messages to the events topic:

```bash
for i in {1..10}; do
  echo "{\"eventType\": \"click\", \"userId\": \"user-$i\", \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
done | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic events --bootstrap-server localhost:9092
```

### Task 6: Wait for Backup

The connector will flush data when:
- `flush.size` (5) records are reached, OR
- `rotate.interval.ms` (60 seconds) has elapsed

Wait about 60 seconds for the backup to occur:

```bash
sleep 65
```

### Task 7: Verify Backup Files in MinIO

List all backed-up files:

```bash
docker exec minio mc ls --recursive minio/kafka-backup/topics/
```

You should see files like:
```
topics/orders/partition=0/orders+0+0000000000.json.gz
topics/events/partition=0/events+0+0000000000.json.gz
```

### Task 8: Inspect a Backup File

Download and view the contents of a backup file:

```bash
# List files to find the exact filename
docker exec minio mc ls minio/kafka-backup/topics/orders/partition=0/

# View contents (replace filename as needed)
docker exec minio mc cat minio/kafka-backup/topics/orders/partition=0/orders+0+0000000000.json.gz | gunzip
```

Each line in the file is a JSON record from your topic.

### Task 9: Check Storage Usage

```bash
docker exec minio mc du minio/kafka-backup/
```

### Task 10: Monitor Connector Status

Check the connector's consumer lag:

```bash
docker exec kafka /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group connect-s3-sink \
  --describe
```

Low or zero lag indicates the connector is keeping up with new messages.

---

## Part 2: Topic Recovery

### Task 11: Simulate Data Loss

Delete the orders topic:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --delete --topic orders \
  --bootstrap-server localhost:9092
```

Verify it's gone:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server localhost:9092
```

### Task 12: Recreate the Topic

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic orders-restored \
  --partitions 3 --replication-factor 1 \
  --bootstrap-server localhost:9092
```

### Task 13: Restore from Backup

First, set up the MinIO alias (if not already done):

```bash
docker exec minio mc alias set minio http://localhost:9000 minioadmin minioadmin
```

List available backup files:

```bash
docker exec minio mc ls --recursive minio/kafka-backup/topics/orders/
```

Restore each backup file to Kafka (repeat for each file listed):

```bash
# Example: restore a specific file (adjust the path based on your output)
docker exec minio mc cat minio/kafka-backup/topics/orders/partition=0/orders+0+0000000000.json.gz | \
  gunzip | \
  docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
    --topic orders-restored --bootstrap-server localhost:9092
```

Or restore all files with a loop:

```bash
# List all backup files and restore each one
for file in $(docker exec minio mc ls --recursive minio/kafka-backup/topics/orders/ | awk '{print $NF}'); do
  echo "Restoring: $file"
  docker exec minio mc cat "minio/kafka-backup/topics/orders/$file" | gunzip | \
    docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
      --topic orders-restored --bootstrap-server localhost:9092
done
```

### Task 14: Verify Restoration

Count restored messages:

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic orders-restored \
  --from-beginning \
  --timeout-ms 5000 \
  --bootstrap-server localhost:9092 2>/dev/null | wc -l
```

View restored messages:

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic orders-restored \
  --from-beginning \
  --max-messages 5 \
  --bootstrap-server localhost:9092
```

---

## Key Concepts

### S3 Sink Connector Behavior

1. **Consumes** messages from Kafka topics
2. **Buffers** records in memory
3. **Flushes** to S3 when buffer is full OR time interval reached
4. **Creates** new file for each flush

### File Naming Convention

```
{topic}+{partition}+{startOffset}.{format}.{compression}
```

Example: `orders+0+0000000000.json.gz`
- Topic: orders
- Partition: 0
- Starting offset: 0
- Format: JSON
- Compression: gzip

### Flush Strategies

| Strategy | Setting | Use Case |
|----------|---------|----------|
| Record count | `flush.size=1000` | High-volume topics |
| Time-based | `rotate.interval.ms=60000` | Consistent backup intervals |
| Scheduled | `rotate.schedule.interval.ms=3600000` | Hourly backups |

### Output Formats

| Format | Class | Use Case |
|--------|-------|----------|
| JSON | `io.confluent.connect.s3.format.json.JsonFormat` | Human-readable, debugging |
| Avro | `io.confluent.connect.s3.format.avro.AvroFormat` | Schema evolution, compact |
| Parquet | `io.confluent.connect.s3.format.parquet.ParquetFormat` | Analytics, columnar |

---

## Troubleshooting

### Connector Not Starting

```bash
# Check logs
docker logs kafka-connect | tail -50

# Verify connector was deployed
curl http://localhost:8083/connectors

# Check for errors
curl http://localhost:8083/connectors/s3-sink/status | jq .
```

### No Backup Files Appearing

1. Check if enough messages were produced (need `flush.size` records)
2. Wait for `rotate.interval.ms` to elapse (60 seconds)
3. Check connector status for errors
4. Verify MinIO connectivity

```bash
# Test MinIO from Kafka Connect
docker exec kafka-connect curl -s http://minio:9000
```

### Connector Shows FAILED State

```bash
# Get detailed error
curl http://localhost:8083/connectors/s3-sink/status | jq '.tasks[0].trace'

# Restart the connector
curl -X POST http://localhost:8083/connectors/s3-sink/restart
```

---

## Advanced Configuration

### Time-Based Partitioning

Organize files by date/hour:

```json
{
  "partitioner.class": "io.confluent.connect.storage.partitioner.TimeBasedPartitioner",
  "path.format": "'year'=YYYY/'month'=MM/'day'=dd/'hour'=HH",
  "partition.duration.ms": "3600000",
  "timestamp.extractor": "RecordField",
  "timestamp.field": "timestamp"
}
```

Result: `topics/orders/year=2024/month=01/day=15/hour=10/...`

### Performance Tuning

```json
{
  "tasks.max": "4",
  "s3.part.size": "26214400",
  "flush.size": "10000"
}
```

---

## Cleanup

Stop all services:

```bash
docker compose down -v
```

---

## Best Practices

1. **Set appropriate flush.size**: Balance between file count and recovery granularity
2. **Use compression**: Reduces storage costs significantly
3. **Monitor connector lag**: Alert if lag grows continuously
4. **Test recovery regularly**: Verify backups are actually restorable
5. **Use time-based partitioning**: Easier to find and restore specific time ranges
6. **Enable error logging**: Set `errors.log.enable=true` for debugging

---

## Additional Resources

- [S3 Sink Connector Documentation](https://docs.confluent.io/kafka-connectors/s3-sink/current/)
- [Kafka Connect REST API](https://docs.confluent.io/platform/current/connect/references/restapi.html)
- [MinIO Documentation](https://min.io/docs/minio/linux/index.html)