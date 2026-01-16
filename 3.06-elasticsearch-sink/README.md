# Exercise 3.06: Elasticsearch Sink

## Learning Objectives

- Configure Kafka Connect to sink data to Elasticsearch
- Understand Elasticsearch index mapping and document routing
- Use Single Message Transforms (SMTs) for data preparation
- Monitor connector health and throughput
- Handle schema evolution and index management

## Background

This exercise completes the log broker pipeline: **Sources → Kafka → Elasticsearch**. This is the exact pattern needed for a log broker that ingests data from various sources and delivers it to an Elastic platform for search and analysis.

### Why Kafka → Elasticsearch?

1. **Decoupling**: Log sources don't need direct Elasticsearch access
2. **Buffering**: Kafka handles traffic spikes before Elasticsearch ingestion
3. **Reliability**: Messages persist in Kafka even if Elasticsearch is temporarily down
4. **Replayability**: Reprocess historical data by resetting consumer offsets
5. **Multiple consumers**: Other systems can also consume the same log data

### Elasticsearch Sink Connector

The Elasticsearch Sink Connector reads from Kafka topics and writes documents to Elasticsearch indices. It supports:
- Automatic index creation
- Document ID extraction from message keys or fields
- Schema-based mapping
- Bulk indexing for performance

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Log        │     │              │     │    Kafka     │     │              │
│   Sources    │────▶│    Kafka     │────▶│   Connect    │────▶│ Elasticsearch│
│   (Vector)   │     │              │     │   (Sink)     │     │              │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                                                                      │
                                                                      ▼
                                                               ┌──────────────┐
                                                               │   Kibana     │
                                                               │   (UI)       │
                                                               └──────────────┘
```

## Prerequisites

- Docker and Docker Compose installed
- Completed exercises 3.03 (Kafka Connect) and 3.05 (Vector)

## Setup

Start all services:

```bash
docker compose up -d
```

Wait for all services to be healthy:

```bash
docker compose ps
```

You should see:
- `kafka` - Kafka broker
- `connect` - Kafka Connect with Elasticsearch connector
- `elasticsearch` - Elasticsearch single-node cluster
- `kibana` - Kibana UI for exploring data
- `log-generator` - Produces sample log data to Kafka

## Accessing the Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Kafka Connect | http://localhost:8083 | - |
| Elasticsearch | http://localhost:9200 | - |
| Kibana | http://localhost:5601 | - |
| Kafka UI | http://localhost:8080 | - |

## Tasks

### Task 1: Verify Elasticsearch is Running

Check Elasticsearch health:

```bash
curl http://localhost:9200/_cluster/health?pretty
```

You should see:
```json
{
  "cluster_name" : "docker-cluster",
  "status" : "green",
  ...
}
```

List existing indices:

```bash
curl http://localhost:9200/_cat/indices?v
```

### Task 2: Check Available Kafka Topics

The log generator is producing sample log data:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server localhost:9092
```

Consume some sample messages:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic application-logs \
  --bootstrap-server localhost:9092 \
  --from-beginning \
  --max-messages 5
```

### Task 3: Check Available Connectors

Verify the Elasticsearch connector plugin is installed:

```bash
curl http://localhost:8083/connector-plugins | jq '.[] | select(.class | contains("Elasticsearch"))'
```

### Task 4: Create the Elasticsearch Sink Connector

Create a connector to sink logs to Elasticsearch:

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "elasticsearch-sink",
    "config": {
      "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
      "tasks.max": "1",
      "topics": "application-logs",
      "connection.url": "http://elasticsearch:9200",
      "type.name": "_doc",
      "key.ignore": "true",
      "schema.ignore": "true",
      "value.converter": "org.apache.kafka.connect.json.JsonConverter",
      "value.converter.schemas.enable": "false"
    }
  }'
```

Check connector status:

```bash
curl http://localhost:8083/connectors/elasticsearch-sink/status | jq
```

### Task 5: Verify Data in Elasticsearch

Wait a few seconds for data to flow, then check the index:

```bash
# List indices
curl http://localhost:9200/_cat/indices?v

# Search documents in the application-logs index
curl "http://localhost:9200/application-logs/_search?pretty&size=5"
```

### Task 6: Explore Data in Kibana

1. Open Kibana: http://localhost:5601
2. Go to **Management** → **Stack Management** → **Index Patterns**
3. Create an index pattern: `application-logs*`
4. Select `@timestamp` or `timestamp` as the time field
5. Go to **Discover** to explore the log data

### Task 7: Configure Index Naming by Date

For production log management, create daily indices. Delete and recreate the connector:

```bash
# Delete existing connector
curl -X DELETE http://localhost:8083/connectors/elasticsearch-sink

# Create connector with date-based indices
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "elasticsearch-sink-dated",
    "config": {
      "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
      "tasks.max": "1",
      "topics": "application-logs",
      "connection.url": "http://elasticsearch:9200",
      "type.name": "_doc",
      "key.ignore": "true",
      "schema.ignore": "true",
      "value.converter": "org.apache.kafka.connect.json.JsonConverter",
      "value.converter.schemas.enable": "false",
      "transforms": "routeTS",
      "transforms.routeTS.type": "org.apache.kafka.connect.transforms.TimestampRouter",
      "transforms.routeTS.topic.format": "logs-${topic}-${timestamp}",
      "transforms.routeTS.timestamp.format": "yyyy-MM-dd"
    }
  }'
```

Check for new indices:

```bash
curl http://localhost:9200/_cat/indices?v
```

You should see indices named like `logs-application-logs-2024-01-15`.

### Task 8: Add Field Transformations

Use SMTs to modify data before indexing. Create a connector with additional transforms:

```bash
curl -X DELETE http://localhost:8083/connectors/elasticsearch-sink-dated

curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "elasticsearch-sink-transformed",
    "config": {
      "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
      "tasks.max": "1",
      "topics": "application-logs",
      "connection.url": "http://elasticsearch:9200",
      "type.name": "_doc",
      "key.ignore": "true",
      "schema.ignore": "true",
      "value.converter": "org.apache.kafka.connect.json.JsonConverter",
      "value.converter.schemas.enable": "false",
      "transforms": "addMetadata,routeTS",
      "transforms.addMetadata.type": "org.apache.kafka.connect.transforms.InsertField$Value",
      "transforms.addMetadata.static.field": "indexed_by",
      "transforms.addMetadata.static.value": "kafka-connect",
      "transforms.routeTS.type": "org.apache.kafka.connect.transforms.TimestampRouter",
      "transforms.routeTS.topic.format": "logs-${topic}-${timestamp}",
      "transforms.routeTS.timestamp.format": "yyyy-MM-dd"
    }
  }'
```

Verify the new field appears in documents:

```bash
curl "http://localhost:9200/logs-application-logs-*/_search?pretty&size=1" | grep indexed_by
```

### Task 9: Extract Document ID from Message

For deduplication, use a field from the message as the Elasticsearch document ID:

```bash
curl -X DELETE http://localhost:8083/connectors/elasticsearch-sink-transformed

curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "elasticsearch-sink-with-id",
    "config": {
      "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
      "tasks.max": "1",
      "topics": "application-logs",
      "connection.url": "http://elasticsearch:9200",
      "type.name": "_doc",
      "key.ignore": "true",
      "schema.ignore": "true",
      "value.converter": "org.apache.kafka.connect.json.JsonConverter",
      "value.converter.schemas.enable": "false",
      "transforms": "extractId",
      "transforms.extractId.type": "org.apache.kafka.connect.transforms.ValueToKey",
      "transforms.extractId.fields": "trace_id",
      "key.ignore": "false"
    }
  }'
```

### Task 10: Configure Error Handling

For production, configure proper error handling:

```bash
curl -X PUT http://localhost:8083/connectors/elasticsearch-sink-with-id/config \
  -H "Content-Type: application/json" \
  -d '{
    "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
    "tasks.max": "1",
    "topics": "application-logs",
    "connection.url": "http://elasticsearch:9200",
    "type.name": "_doc",
    "key.ignore": "true",
    "schema.ignore": "true",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": "false",
    "behavior.on.malformed.documents": "warn",
    "behavior.on.null.values": "ignore",
    "max.retries": "5",
    "retry.backoff.ms": "1000",
    "batch.size": "500",
    "linger.ms": "1000",
    "flush.timeout.ms": "30000"
  }'
```

### Task 11: Monitor Connector Metrics

Check connector throughput and errors:

```bash
# Connector status
curl http://localhost:8083/connectors/elasticsearch-sink-with-id/status | jq

# Elasticsearch indexing stats
curl http://localhost:9200/_stats/indexing?pretty

# Documents count
curl http://localhost:9200/application-logs/_count
```

### Task 12: Multiple Topics to Multiple Indices (Optional)

Create a connector that handles multiple topics:

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "elasticsearch-multi-sink",
    "config": {
      "connector.class": "io.confluent.connect.elasticsearch.ElasticsearchSinkConnector",
      "tasks.max": "2",
      "topics": "application-logs,error-logs,audit-logs",
      "connection.url": "http://elasticsearch:9200",
      "type.name": "_doc",
      "key.ignore": "true",
      "schema.ignore": "true",
      "value.converter": "org.apache.kafka.connect.json.JsonConverter",
      "value.converter.schemas.enable": "false"
    }
  }'
```

Each topic automatically creates a corresponding index.

## Verification

You've successfully completed this exercise when:
- ✅ Data flows from Kafka to Elasticsearch
- ✅ You can view logs in Kibana
- ✅ You can configure date-based indices
- ✅ You can apply SMTs for field transformations
- ✅ You understand error handling configuration

## Key Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `connection.url` | Elasticsearch URL | - |
| `type.name` | Document type (use `_doc`) | - |
| `key.ignore` | Ignore message key | true |
| `schema.ignore` | Ignore schema (for schemaless JSON) | false |
| `batch.size` | Docs per bulk request | 2000 |
| `max.retries` | Retry attempts on failure | 5 |
| `behavior.on.malformed.documents` | Action on bad docs: ignore, warn, fail | fail |

## Index Lifecycle Management

For production, configure Elasticsearch ILM (Index Lifecycle Management):

```bash
# Create ILM policy
curl -X PUT "http://localhost:9200/_ilm/policy/logs-policy" \
  -H "Content-Type: application/json" \
  -d '{
    "policy": {
      "phases": {
        "hot": {
          "min_age": "0ms",
          "actions": {
            "rollover": {
              "max_size": "50gb",
              "max_age": "1d"
            }
          }
        },
        "delete": {
          "min_age": "30d",
          "actions": {
            "delete": {}
          }
        }
      }
    }
  }'
```

## Best Practices

1. **Use date-based indices**: Easier retention management and better performance
2. **Configure batching**: Tune `batch.size` and `linger.ms` for throughput
3. **Set up ILM**: Automate index rollover and deletion
4. **Monitor lag**: Track consumer lag to detect backpressure
5. **Use dead letter queue**: For handling poison messages
6. **Index templates**: Define mappings before connector creates indices

## Troubleshooting

### Connector in FAILED state

```bash
# Check error message
curl http://localhost:8083/connectors/elasticsearch-sink/status | jq '.tasks[0].trace'
```

### Documents not appearing

```bash
# Check Elasticsearch logs
docker compose logs elasticsearch

# Verify index exists
curl http://localhost:9200/_cat/indices?v
```

### Mapping conflicts

- Define an index template before starting the connector
- Use `schema.ignore=true` for schemaless JSON

### High latency

- Increase `batch.size` for higher throughput
- Add more tasks with `tasks.max`
- Check Elasticsearch cluster health

## Cleanup

Delete connectors:

```bash
curl -X DELETE http://localhost:8083/connectors/elasticsearch-sink
curl -X DELETE http://localhost:8083/connectors/elasticsearch-sink-dated
curl -X DELETE http://localhost:8083/connectors/elasticsearch-sink-transformed
curl -X DELETE http://localhost:8083/connectors/elasticsearch-sink-with-id
curl -X DELETE http://localhost:8083/connectors/elasticsearch-multi-sink
```

Stop and remove all containers:

```bash
docker compose down -v
```

## Additional Resources

- [Confluent Elasticsearch Sink Connector](https://docs.confluent.io/kafka-connectors/elasticsearch/current/overview.html)
- [Elasticsearch Index Templates](https://www.elastic.co/guide/en/elasticsearch/reference/current/index-templates.html)
- [Index Lifecycle Management](https://www.elastic.co/guide/en/elasticsearch/reference/current/index-lifecycle-management.html)
- [Kafka Connect Transforms](https://docs.confluent.io/platform/current/connect/transforms/overview.html)

## Next Steps

Continue to [Exercise 4.01: Event Design](../4.01-event-design/)