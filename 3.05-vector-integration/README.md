# Exercise 3.05: Vector Integration

## Learning Objectives

- Understand Vector as a log/metrics/traces pipeline tool
- Configure Vector to collect logs and send them to Kafka
- Use Vector as a Kafka source (reading from Kafka)
- Transform and route log data through Vector pipelines
- Build a complete log collection pipeline for observability

## Background

[Vector](https://vector.dev/) is a high-performance observability data pipeline that can collect, transform, and route logs, metrics, and traces. It's commonly used in modern infrastructure as an alternative to tools like Logstash, Fluentd, or Filebeat.

### Why Vector with Kafka?

In a log broker architecture, Vector serves as the agent that:
1. **Collects** logs from various sources (files, journald, Docker, etc.)
2. **Transforms** log data (parsing, enrichment, filtering)
3. **Routes** logs to Kafka as a central message broker
4. **Consumes** from Kafka to forward to final destinations

This pattern is exactly what Universiteit Leiden needs: logs from various sources → Kafka → Elastic.

### Vector Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Vector Pipeline                            │
│                                                                     │
│  ┌─────────┐    ┌──────────────┐    ┌─────────┐                    │
│  │ Sources │───▶│  Transforms  │───▶│  Sinks  │                    │
│  └─────────┘    └──────────────┘    └─────────┘                    │
│                                                                     │
│  Examples:       Examples:           Examples:                      │
│  - file          - remap             - kafka                        │
│  - journald      - filter            - elasticsearch                │
│  - docker_logs   - parse_json        - console                      │
│  - kafka         - add_fields        - file                         │
│  - syslog        - dedupe            - prometheus                   │
└─────────────────────────────────────────────────────────────────────┘
```

## Architecture for This Exercise

```
┌────────────────┐     ┌────────────────┐     ┌────────────────┐
│  Log Generator │     │                │     │                │
│  (demo-app)    │────▶│  Vector Agent  │────▶│     Kafka      │
│                │     │  (collector)   │     │                │
└────────────────┘     └────────────────┘     └────────────────┘
                                                      │
                                                      ▼
                       ┌────────────────┐     ┌────────────────┐
                       │    Console     │◀────│  Vector Agent  │
                       │   (output)     │     │  (consumer)    │
                       └────────────────┘     └────────────────┘
```

## Prerequisites

- Docker and Docker Compose installed
- Completed exercises 3.01 through 3.04

## Setup

Start all services:

```bash
docker compose up -d
```

Verify services are running:

```bash
docker compose ps
```

You should see:
- `kafka` - Kafka broker
- `vector-collector` - Vector collecting logs
- `vector-consumer` - Vector consuming from Kafka
- `demo-app` - Application generating sample logs

## Tasks

### Task 1: Understand the Vector Configuration

Vector uses a TOML configuration file. Let's examine the collector configuration:

```bash
cat config/vector-collector.toml
```

Key sections:
- **`[sources]`**: Where data comes from
- **`[transforms]`**: How data is processed
- **`[sinks]`**: Where data goes

### Task 2: Observe Log Collection

The demo-app container generates sample log data. View the raw logs:

```bash
docker compose logs -f demo-app
```

Now check what Vector is sending to Kafka:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic application-logs \
  --bootstrap-server localhost:9092 \
  --from-beginning
```

Press Ctrl+C to stop.

### Task 3: Examine the Log Structure

Vector transforms logs into structured JSON. Check the Kafka topic for formatted data:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic application-logs \
  --bootstrap-server localhost:9092 \
  --from-beginning \
  --max-messages 5
```

Notice the structure:
- `timestamp`: When the log was generated
- `host`: Source hostname
- `source_type`: Type of Vector source
- `message`: The actual log message
- Additional fields added by transforms

### Task 4: Add Custom Fields with VRL

Vector uses **VRL (Vector Remap Language)** for transformations. Let's modify the configuration to add custom fields.

Edit `config/vector-collector.toml` and add a new transform:

```toml
[transforms.enrich]
type = "remap"
inputs = ["parse_logs"]
source = '''
  .environment = "training"
  .team = "observability"
  .processed_at = now()
'''
```

Then update the sink to use the new transform:

```toml
[sinks.kafka]
inputs = ["enrich"]  # Changed from "parse_logs"
```

Restart the collector to apply changes:

```bash
docker compose restart vector-collector
```

Verify the new fields appear:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic application-logs \
  --bootstrap-server localhost:9092 \
  --max-messages 3
```

### Task 5: Filter Logs by Severity

Add a filter to only send ERROR and WARN logs to a separate topic:

Edit `config/vector-collector.toml`:

```toml
[transforms.filter_errors]
type = "filter"
inputs = ["enrich"]
condition = '.level == "ERROR" || .level == "WARN"'

[sinks.kafka_errors]
type = "kafka"
inputs = ["filter_errors"]
bootstrap_servers = "kafka:9092"
topic = "error-logs"
encoding.codec = "json"
```

Restart and verify:

```bash
docker compose restart vector-collector

# Check the new topic
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server localhost:9092

docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic error-logs \
  --bootstrap-server localhost:9092 \
  --from-beginning
```

### Task 6: Route Logs by Application

Create different topics based on application name:

```toml
[sinks.kafka_routed]
type = "kafka"
inputs = ["enrich"]
bootstrap_servers = "kafka:9092"
topic = "logs-{{ .app_name }}"
encoding.codec = "json"
```

This uses templating to dynamically set the topic based on the `app_name` field.

### Task 7: Check Vector Metrics

Vector exposes internal metrics. Check them:

```bash
# Vector exposes Prometheus metrics on port 9598
curl http://localhost:9598/metrics | grep vector_
```

Key metrics to monitor:
- `vector_component_received_events_total`: Events received per component
- `vector_component_sent_events_total`: Events sent per component
- `vector_buffer_events`: Events in buffer

### Task 8: Test High-Volume Ingestion

Generate more logs to test throughput:

```bash
# Run a load generator
docker compose exec demo-app /bin/sh -c '
  for i in $(seq 1 1000); do
    echo "{\"timestamp\":\"$(date -Iseconds)\",\"level\":\"INFO\",\"app_name\":\"loadtest\",\"message\":\"Load test message $i\"}"
    sleep 0.01
  done
' >> /var/log/app/application.log
```

Monitor Vector's throughput:

```bash
# Check events processed
curl -s http://localhost:9598/metrics | grep "vector_component_sent_events_total"
```

### Task 9: Configure Batching and Buffering

For production, configure proper batching to optimize Kafka writes.

Edit `config/vector-collector.toml`:

```toml
[sinks.kafka]
type = "kafka"
inputs = ["enrich"]
bootstrap_servers = "kafka:9092"
topic = "application-logs"
encoding.codec = "json"

# Batching configuration
batch.max_bytes = 1048576      # 1 MB max batch size
batch.timeout_secs = 1         # Flush every second

# Buffer configuration
buffer.type = "disk"
buffer.max_size = 268435488    # 256 MB disk buffer
buffer.when_full = "block"
```

### Task 10: Vector as Kafka Consumer

Check the consumer Vector instance that reads from Kafka:

```bash
cat config/vector-consumer.toml
```

View its output:

```bash
docker compose logs -f vector-consumer
```

This demonstrates Vector's ability to also consume from Kafka and forward to other destinations.

### Task 11: Parse Syslog Format (Optional)

Vector can parse various log formats. Add syslog parsing:

```toml
[sources.syslog_input]
type = "syslog"
address = "0.0.0.0:514"
mode = "udp"

[transforms.parse_syslog]
type = "remap"
inputs = ["syslog_input"]
source = '''
  . = parse_syslog!(.message)
  .source = "syslog"
'''

[sinks.kafka_syslog]
type = "kafka"
inputs = ["parse_syslog"]
bootstrap_servers = "kafka:9092"
topic = "syslog-logs"
encoding.codec = "json"
```

### Task 12: Handle Parse Errors (Optional)

Add error handling for malformed logs:

```toml
[transforms.parse_logs]
type = "remap"
inputs = ["file_source"]
source = '''
  parsed, err = parse_json(.message)
  if err != null {
    .parse_error = err
    .original_message = .message
    .level = "PARSE_ERROR"
  } else {
    . = merge(., parsed)
  }
'''
```

## Verification

You've successfully completed this exercise when:
- ✅ Logs are flowing from the demo-app to Kafka via Vector
- ✅ You can add custom fields with VRL transforms
- ✅ You can filter logs based on conditions
- ✅ You understand Vector's source → transform → sink pipeline
- ✅ You can route logs to different topics dynamically

## Key Vector Concepts

| Concept | Description |
|---------|-------------|
| **Source** | Where data enters Vector (file, kafka, syslog, etc.) |
| **Transform** | Modify, filter, or route data (remap, filter, route) |
| **Sink** | Where data exits Vector (kafka, elasticsearch, console) |
| **VRL** | Vector Remap Language for transformations |
| **Buffer** | Temporary storage for reliability (memory or disk) |

## VRL Quick Reference

```vrl
# Parse JSON
. = parse_json!(.message)

# Add field
.environment = "production"

# Delete field
del(.unwanted_field)

# Conditional
if .level == "ERROR" {
  .alert = true
}

# Parse timestamp
.timestamp = parse_timestamp!(.timestamp, "%Y-%m-%d %H:%M:%S")

# String manipulation
.message = downcase(.message)
```

## Best Practices

1. **Use disk buffers in production**: Prevents data loss during outages
2. **Batch writes to Kafka**: Improves throughput significantly
3. **Parse early**: Transform logs at collection time
4. **Add source metadata**: Include host, source_type, and timestamps
5. **Handle parse errors**: Don't drop malformed logs
6. **Monitor Vector metrics**: Track throughput and errors

## Troubleshooting

### Vector not starting

```bash
# Check configuration syntax
docker compose exec vector-collector vector validate /etc/vector/vector.toml
```

### Logs not appearing in Kafka

```bash
# Check Vector logs
docker compose logs vector-collector

# Verify Kafka connectivity
docker compose exec vector-collector nc -zv kafka 9092
```

### High memory usage

- Switch from memory buffer to disk buffer
- Reduce batch sizes
- Check for slow sinks causing backpressure

## Cleanup

Stop all containers:

```bash
docker compose down
```

Remove all data:

```bash
docker compose down -v
```

## Additional Resources

- [Vector Documentation](https://vector.dev/docs/)
- [VRL Reference](https://vector.dev/docs/reference/vrl/)
- [Vector Kafka Sink](https://vector.dev/docs/reference/configuration/sinks/kafka/)
- [Vector Kafka Source](https://vector.dev/docs/reference/configuration/sources/kafka/)

## Next Steps

Continue to [Exercise 3.06: Elasticsearch Sink](../3.06-elasticsearch-sink/)