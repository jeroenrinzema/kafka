# Exercise 3.03: Kafka Connect Fundamentals

## Learning Objectives

- Understand Kafka Connect architecture (workers, connectors, tasks)
- Run Kafka Connect in distributed mode
- Manage connectors using the REST API
- Create source connectors to ingest data into Kafka
- Create sink connectors to export data from Kafka
- Configure error handling and Dead Letter Queues
- Apply Single Message Transforms (SMTs)

## Background

Kafka Connect is a framework for streaming data between Apache Kafka and other systems. It provides a scalable, reliable way to move data without writing custom code.

### Key Concepts

| Concept | Description |
|---------|-------------|
| **Connector** | A logical job that defines how data should be copied (source or sink) |
| **Task** | The actual worker that performs the data movement (connectors can have multiple tasks) |
| **Worker** | The JVM process that runs connectors and tasks |
| **Source Connector** | Pulls data FROM external systems INTO Kafka |
| **Sink Connector** | Pushes data FROM Kafka TO external systems |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kafka Connect Cluster                        │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Worker 1   │  │   Worker 2   │  │   Worker 3   │          │
│  │              │  │              │  │              │          │
│  │ ┌──────────┐ │  │ ┌──────────┐ │  │ ┌──────────┐ │          │
│  │ │ Task 1-1 │ │  │ │ Task 1-2 │ │  │ │ Task 2-1 │ │          │
│  │ └──────────┘ │  │ └──────────┘ │  │ └──────────┘ │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│         │                 │                 │                   │
│         └─────────────────┼─────────────────┘                   │
│                           ▼                                     │
│                    ┌─────────────┐                              │
│                    │    Kafka    │                              │
│                    │   Cluster   │                              │
│                    └─────────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

### Standalone vs Distributed Mode

| Mode | Use Case | Fault Tolerance |
|------|----------|-----------------|
| **Standalone** | Development, single-node testing | None - single point of failure |
| **Distributed** | Production deployments | Yes - tasks redistributed on failure |

This exercise uses **distributed mode**, which is what you'll use in production.

## Prerequisites

- Docker and Docker Compose installed
- Completed exercises 1.01-1.06 (Kafka fundamentals)
- Basic understanding of REST APIs

## Setup

Start the Kafka Connect cluster:

```bash
docker compose up -d
```

Wait about 30 seconds for all services to start. Check that Connect is ready:

```bash
curl -s http://localhost:8083/ | jq
```

You should see the Connect version information.

Verify the cluster has one worker:

```bash
curl -s http://localhost:8083/connectors | jq
```

This should return an empty array `[]` since we haven't created any connectors yet.

## Tasks

### Task 1: Explore the REST API

Kafka Connect exposes a REST API for managing connectors. Let's explore it.

**List available connector plugins:**

```bash
curl -s http://localhost:8083/connector-plugins | jq
```

You should see the built-in connectors:
- `FileStreamSourceConnector` - reads from files
- `FileStreamSinkConnector` - writes to files

**Check cluster status:**

```bash
curl -s http://localhost:8083/ | jq
```

**Key REST endpoints:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/connectors` | List all connectors |
| POST | `/connectors` | Create a new connector |
| GET | `/connectors/{name}` | Get connector details |
| GET | `/connectors/{name}/status` | Get connector status |
| PUT | `/connectors/{name}/config` | Update connector config |
| POST | `/connectors/{name}/restart` | Restart a connector |
| PUT | `/connectors/{name}/pause` | Pause a connector |
| PUT | `/connectors/{name}/resume` | Resume a connector |
| DELETE | `/connectors/{name}` | Delete a connector |

### Task 2: Create a Source Connector

Create a FileStreamSource connector to read log lines from a file into Kafka:

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "file-source",
    "config": {
      "connector.class": "org.apache.kafka.connect.file.FileStreamSourceConnector",
      "tasks.max": "1",
      "file": "/data/input/server.log",
      "topic": "server-logs"
    }
  }' | jq
```

**Verify the connector is running:**

```bash
curl -s http://localhost:8083/connectors/file-source/status | jq
```

You should see:
```json
{
  "name": "file-source",
  "connector": {
    "state": "RUNNING",
    "worker_id": "connect:8083"
  },
  "tasks": [
    {
      "id": 0,
      "state": "RUNNING",
      "worker_id": "connect:8083"
    }
  ]
}
```

### Task 3: Verify Data in Kafka

Check that the log lines are in Kafka:

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic server-logs \
  --from-beginning \
  --max-messages 5
```

You should see the log lines from the `server.log` file.

**Check topic details:**

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic server-logs
```

### Task 4: Create a Sink Connector

Now create a FileStreamSink connector to write data from Kafka to a file:

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "file-sink",
    "config": {
      "connector.class": "org.apache.kafka.connect.file.FileStreamSinkConnector",
      "tasks.max": "1",
      "file": "/data/output/processed.log",
      "topics": "server-logs"
    }
  }' | jq
```

**Verify the connector is running:**

```bash
curl -s http://localhost:8083/connectors/file-sink/status | jq
```

**Check the output file:**

```bash
cat connect-data/output/processed.log
```

You should see the same log lines that were read from the input file!

### Task 5: Append Data to the Source File

The FileStreamSource connector monitors the file for new content. Let's add more data:

```bash
echo "2025-01-15 10:05:00 INFO  [http-nio-8080] POST /api/orders - 201 Created (55ms)" >> connect-data/input/server.log
echo "2025-01-15 10:05:01 DEBUG [order-service] Order created: order_id=1004, customer=eve@example.com, amount=199.99" >> connect-data/input/server.log
```

Wait a few seconds, then check the output file:

```bash
tail -5 connect-data/output/processed.log
```

The new lines should appear! This demonstrates real-time streaming.

### Task 6: List and Inspect Connectors

**List all connectors:**

```bash
curl -s http://localhost:8083/connectors | jq
```

**Get connector configuration:**

```bash
curl -s http://localhost:8083/connectors/file-source/config | jq
```

**Get detailed connector info:**

```bash
curl -s http://localhost:8083/connectors/file-source | jq
```

### Task 7: Pause and Resume a Connector

**Pause the sink connector:**

```bash
curl -X PUT http://localhost:8083/connectors/file-sink/pause
```

**Check the status:**

```bash
curl -s http://localhost:8083/connectors/file-sink/status | jq
```

The state should now be `PAUSED`.

**Add more data while paused:**

```bash
echo "2025-01-15 10:06:00 WARN  [memory-monitor] Memory usage at 80%" >> connect-data/input/server.log
```

**Check that the output file hasn't been updated:**

```bash
tail -3 connect-data/output/processed.log
```

**Resume the connector:**

```bash
curl -X PUT http://localhost:8083/connectors/file-sink/resume
```

Wait a moment, then check the output file again:

```bash
tail -3 connect-data/output/processed.log
```

The new line should now appear!

### Task 8: Update Connector Configuration

You can update a connector's configuration without deleting and recreating it:

```bash
curl -X PUT http://localhost:8083/connectors/file-source/config \
  -H "Content-Type: application/json" \
  -d '{
    "connector.class": "org.apache.kafka.connect.file.FileStreamSourceConnector",
    "tasks.max": "1",
    "file": "/data/input/server.log",
    "topic": "server-logs-v2"
  }' | jq
```

This changes the target topic to `server-logs-v2`.

**Verify the new topic is being used:**

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list | grep server-logs
```

You should see both `server-logs` and `server-logs-v2`.

**Revert to the original topic:**

```bash
curl -X PUT http://localhost:8083/connectors/file-source/config \
  -H "Content-Type: application/json" \
  -d '{
    "connector.class": "org.apache.kafka.connect.file.FileStreamSourceConnector",
    "tasks.max": "1",
    "file": "/data/input/server.log",
    "topic": "server-logs"
  }' | jq
```

### Task 9: Apply Single Message Transforms (SMTs)

SMTs allow you to modify messages as they pass through Connect. Let's add metadata fields to each message.

> **Note:** The `FileStreamSourceConnector` produces plain string values (not structured records). To add fields, we first need to wrap the string in a struct using `HoistField`, then we can use `InsertField` to add additional fields.

**Update the source connector with SMTs:**

```bash
curl -X PUT http://localhost:8083/connectors/file-source/config \
  -H "Content-Type: application/json" \
  -d '{
    "connector.class": "org.apache.kafka.connect.file.FileStreamSourceConnector",
    "tasks.max": "1",
    "file": "/data/input/server.log",
    "topic": "server-logs",
    "transforms": "hoistToStruct,addTimestamp,addSource",
    "transforms.hoistToStruct.type": "org.apache.kafka.connect.transforms.HoistField$Value",
    "transforms.hoistToStruct.field": "line",
    "transforms.addTimestamp.type": "org.apache.kafka.connect.transforms.InsertField$Value",
    "transforms.addTimestamp.timestamp.field": "ingested_at",
    "transforms.addSource.type": "org.apache.kafka.connect.transforms.InsertField$Value",
    "transforms.addSource.static.field": "source_system",
    "transforms.addSource.static.value": "webserver-01"
  }' | jq
```

This transform chain does the following:
1. **hoistToStruct**: Wraps the plain string value in a struct: `"log line"` → `{"line": "log line"}`
2. **addTimestamp**: Adds the ingestion timestamp: `{"line": "...", "ingested_at": 1234567890}`
3. **addSource**: Adds a static source field: `{"line": "...", "ingested_at": ..., "source_system": "webserver-01"}`

**Add a new line to trigger the transform:**

```bash
echo "2025-01-15 10:07:00 INFO  [http-nio-8080] GET /api/status - 200 OK (5ms)" >> connect-data/input/server.log
```

**Check the transformed message in Kafka:**

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic server-logs \
  --from-beginning \
  --max-messages 5
```

You should see JSON objects like:
```json
{"line":"2025-01-15 10:00:01 INFO  [main] Application started successfully","ingested_at":1736942400000,"source_system":"webserver-01"}
```

**Common SMTs:**

| Transform | Purpose |
|-----------|---------|
| `HoistField` | Wrap a primitive value in a struct (essential for working with string data) |
| `InsertField` | Add static or dynamic fields |
| `ReplaceField` | Rename, include, or exclude fields |
| `MaskField` | Mask sensitive data |
| `TimestampConverter` | Convert timestamp formats |
| `ExtractField` | Extract a field from a struct |
| `Filter` | Drop messages based on conditions |
| `RegexRouter` | Route to different topics based on regex |

> **Tip:** SMT order matters! Transforms are applied in the order listed. When working with primitive values (like strings from FileStreamSource), always use `HoistField` first before applying `InsertField` or other struct-based transforms.

### Task 10: Configure Error Handling and Dead Letter Queue

Create a new connector with DLQ configuration:

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "file-sink-with-dlq",
    "config": {
      "connector.class": "org.apache.kafka.connect.file.FileStreamSinkConnector",
      "tasks.max": "1",
      "file": "/data/output/processed-with-dlq.log",
      "topics": "server-logs",
      "errors.tolerance": "all",
      "errors.log.enable": "true",
      "errors.log.include.messages": "true",
      "errors.deadletterqueue.topic.name": "server-logs-dlq",
      "errors.deadletterqueue.topic.replication.factor": "1",
      "errors.deadletterqueue.context.headers.enable": "true"
    }
  }' | jq
```

**Error handling options:**

| Setting | Values | Description |
|---------|--------|-------------|
| `errors.tolerance` | `none`, `all` | `none` = fail on error, `all` = skip and continue |
| `errors.log.enable` | `true`, `false` | Log errors to Connect worker log |
| `errors.log.include.messages` | `true`, `false` | Include the problematic message in logs |
| `errors.deadletterqueue.topic.name` | topic name | Send failed messages to this topic |

### Task 11: Restart a Failed Connector

If a connector fails, you can restart it:

**Restart the connector:**

```bash
curl -X POST http://localhost:8083/connectors/file-source/restart
```

**Restart a specific task:**

```bash
curl -X POST http://localhost:8083/connectors/file-source/tasks/0/restart
```

**Check task status:**

```bash
curl -s http://localhost:8083/connectors/file-source/tasks | jq
```

### Task 12: Delete a Connector

When you're done with a connector, delete it:

```bash
curl -X DELETE http://localhost:8083/connectors/file-sink-with-dlq
```

**Verify it's gone:**

```bash
curl -s http://localhost:8083/connectors | jq
```

### Task 13: View in Kafka UI

Open http://localhost:8080 in your browser.

1. Navigate to **Kafka Connect** in the left menu
2. You should see the `connect` cluster
3. Click on it to see the running connectors
4. Explore the connector configurations and status

### Task 14: Monitor Connect Internal Topics

Connect uses internal topics to store configuration and state:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list | grep connect
```

You should see:
- `connect-configs` - Connector configurations
- `connect-offsets` - Source connector offsets
- `connect-status` - Connector and task status

**Peek at the config topic:**

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic connect-configs \
  --from-beginning \
  --property print.key=true \
  --max-messages 5
```

## Verification

You've successfully completed this exercise when you can:

- ✅ Create source connectors using the REST API
- ✅ Create sink connectors using the REST API
- ✅ List and inspect connector status
- ✅ Pause and resume connectors
- ✅ Update connector configurations
- ✅ Apply Single Message Transforms
- ✅ Configure error handling and DLQ
- ✅ Delete connectors
- ✅ Understand Connect's internal topics

## Key Concepts

### Connector Lifecycle

```
Created → Running → Paused → Running → Deleted
              │         ↑
              └─ Failed ┘
                   │
                   └── Restart
```

### Offset Management

- **Source connectors**: Store offsets in `connect-offsets` topic
- **Sink connectors**: Use Kafka consumer offsets (standard consumer group)
- Offsets survive connector restarts and worker failures

### Scaling Connectors

You can scale a connector by increasing `tasks.max`:

```json
{
  "tasks.max": "3"
}
```

Each task runs in parallel, processing different partitions or file segments.

## Cleanup

Stop all services:

```bash
docker compose down -v
```

Remove generated files:

```bash
rm -rf connect-data/output/*
```

## Troubleshooting

### Connector stuck in FAILED state

Check the Connect worker logs:

```bash
docker logs connect 2>&1 | tail -50
```

Check the connector status for error details:

```bash
curl -s http://localhost:8083/connectors/file-source/status | jq '.tasks[0].trace'
```

### No data appearing in Kafka

1. Verify the source file exists and has content
2. Check connector status is `RUNNING`
3. Verify the topic name matches
4. Check Connect worker logs for errors

### REST API not responding

Wait for Connect to fully start (can take 30+ seconds):

```bash
docker logs connect 2>&1 | grep -i "Kafka Connect started"
```

## Best Practices

1. **Use descriptive connector names** - Makes monitoring easier
2. **Set appropriate tasks.max** - Match parallelism to your data
3. **Enable error handling** - Use DLQ in production
4. **Monitor connector status** - Alert on FAILED state
5. **Use SMTs sparingly** - Complex transforms should be in stream processing
6. **Test configuration changes** - Use pause/resume during updates
7. **Version your connector configs** - Store in source control

## Additional Resources

- [Kafka Connect Documentation](https://kafka.apache.org/documentation/#connect)
- [Connect REST API Reference](https://kafka.apache.org/documentation/#connect_rest)
- [Single Message Transforms](https://kafka.apache.org/documentation/#connect_transforms)
- [Connect Configurations](https://kafka.apache.org/documentation/#connectconfigs)

## Next Steps

Continue to [Exercise 3.04: Change Data Capture with Debezium](../3.04-debezium-cdc/) to learn how to capture database changes as Kafka events.
