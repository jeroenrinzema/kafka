# Exercise 19: ksqlDB Basics

## Learning Objectives

- Understand what ksqlDB is and its use cases
- Create streams and tables in ksqlDB
- Write queries to filter and transform data
- Perform aggregations and joins
- Create materialized views

## Background

ksqlDB is a database purpose-built for stream processing applications. It allows you to process Kafka data using SQL-like queries without writing code. ksqlDB supports:

- **Streams**: Immutable, append-only collections (like Kafka topics)
- **Tables**: Mutable collections that represent the latest state
- **Queries**: Transformations and aggregations on streams/tables
- **Materialized Views**: Continuously updated query results

## Setup

1. Start the Kafka broker and ksqlDB:
```bash
docker compose up -d
```

2. Wait for ksqlDB server to be ready (about 10-15 seconds). You can check the logs:
```bash
docker logs ksqldb-server --follow
```

Wait until you see: `Server up and running`

Press `Ctrl+C` to stop following logs.

3. Connect to the ksqlDB CLI:
```bash
docker exec -it ksqldb-cli ksql http://ksqldb-server:8088 2>/dev/null
```

You should see the ksqlDB prompt:
```
ksql>
```

**Note**: The `2>/dev/null` redirects Java warning messages to suppress harmless JMX logging. If you prefer to see all output, omit this part.

## Tasks

### Task 1: Create Your First Stream

First, create the topic:

```bash
docker exec -it broker /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --topic user-events \
  --partitions 3
```

Now in the ksqlDB CLI, create a stream from this topic:

```sql
CREATE STREAM user_events_stream (
  user_id VARCHAR,
  event_type VARCHAR,
  timestamp VARCHAR
) WITH (
  KAFKA_TOPIC='user-events',
  VALUE_FORMAT='JSON'
);
```

**Important**: ksqlDB streams only process messages that arrive **after** the stream is created.

Now, let's produce some sample data. In another terminal, produce some user events:

```bash
docker exec -it broker /opt/kafka/bin/kafka-console-producer.sh \
  --bootstrap-server localhost:9092 \
  --topic user-events \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Send these messages:
```
user1:{"user_id":"user1","event_type":"login","timestamp":"2025-12-08T10:00:00"}
user2:{"user_id":"user2","event_type":"login","timestamp":"2025-12-08T10:05:00"}
user1:{"user_id":"user1","event_type":"page_view","timestamp":"2025-12-08T10:10:00"}
user3:{"user_id":"user3","event_type":"login","timestamp":"2025-12-08T10:15:00"}
user1:{"user_id":"user1","event_type":"purchase","timestamp":"2025-12-08T10:20:00"}
```

Leave the producer terminal open for now (don't press `Ctrl+C` yet).

**Question**: What does this stream represent?

### Task 2: Query the Stream

View all events in real-time:
```sql
SELECT * FROM user_events_stream EMIT CHANGES;
```

This is a **push query** - it continuously outputs new data. Press `Ctrl+C` to stop.

**Note**: You should see the events you just produced. If you don't see anything, send more messages in the producer terminal.

Filter events by type:
```sql
SELECT user_id, timestamp 
FROM user_events_stream 
WHERE event_type = 'login'
EMIT CHANGES;
```

Now send some login events in the producer terminal to see them appear:
```
user4:{"user_id":"user4","event_type":"login","timestamp":"2025-12-08T11:00:00"}
user5:{"user_id":"user5","event_type":"login","timestamp":"2025-12-08T11:05:00"}
```

**Tip**: To read historical data from the beginning of the topic, set the offset:
```sql
SET 'auto.offset.reset' = 'earliest';
SELECT * FROM user_events_stream EMIT CHANGES;
```

This will replay all messages from the topic start.

### Task 3: Create a Derived Stream

Create a new stream with only purchase events:

```sql
CREATE STREAM purchases AS
  SELECT user_id, timestamp
  FROM user_events_stream
  WHERE event_type = 'purchase'
  EMIT CHANGES;
```

Check the underlying topic:
```sql
SHOW TOPICS;
```

Verify the stream has data:
```sql
SELECT * FROM purchases EMIT CHANGES;
```

### Task 4: Aggregations and Tables

Count events per user. First, add more events in the producer terminal:

```
user1:{"user_id":"user1","event_type":"page_view","timestamp":"2025-12-08T10:25:00"}
user2:{"user_id":"user2","event_type":"page_view","timestamp":"2025-12-08T10:30:00"}
user2:{"user_id":"user2","event_type":"purchase","timestamp":"2025-12-08T10:35:00"}
user3:{"user_id":"user3","event_type":"page_view","timestamp":"2025-12-08T10:40:00"}
```

Now create a table with event counts:

```sql
CREATE TABLE user_event_counts AS
  SELECT user_id,
         COUNT(*) AS event_count
  FROM user_events_stream
  GROUP BY user_id
  EMIT CHANGES;
```

Query the table (a **pull query** for current state):
```sql
SELECT * FROM user_event_counts;
```

Or watch it update in real-time:
```sql
SELECT * FROM user_event_counts EMIT CHANGES;
```

### Task 5: Filtering and Transformations

Create a stream with enhanced data:

```sql
CREATE STREAM enriched_events AS
  SELECT user_id,
         event_type,
         timestamp,
         CASE 
           WHEN event_type = 'purchase' THEN 'HIGH'
           WHEN event_type = 'login' THEN 'MEDIUM'
           ELSE 'LOW'
         END AS priority
  FROM user_events_stream
  EMIT CHANGES;
```

View the enriched stream:
```sql
SELECT * FROM enriched_events EMIT CHANGES;
```

### Task 6: Window Aggregations

Count events in 5-minute windows. First, we need to use proper timestamps:

```sql
CREATE STREAM user_events_with_timestamp (
  user_id VARCHAR,
  event_type VARCHAR,
  timestamp VARCHAR
) WITH (
  KAFKA_TOPIC='user-events',
  VALUE_FORMAT='JSON',
  TIMESTAMP='timestamp',
  TIMESTAMP_FORMAT='yyyy-MM-dd''T''HH:mm:ss'
);
```

Create a windowed aggregation:
```sql
CREATE TABLE events_per_window AS
  SELECT user_id,
         COUNT(*) AS event_count,
         WINDOWSTART AS window_start,
         WINDOWEND AS window_end
  FROM user_events_with_timestamp
  WINDOW TUMBLING (SIZE 5 MINUTES)
  GROUP BY user_id
  EMIT CHANGES;
```

### Task 7: Joins

Create a user profile topic with the same number of partitions as the events topic:

```bash
docker exec -it broker /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --topic user-profiles \
  --partitions 3
```

In the producer terminal, send profile data:

```bash
docker exec -it broker /opt/kafka/bin/kafka-console-producer.sh \
  --bootstrap-server localhost:9092 \
  --topic user-profiles \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Send profile data:
```
user1:{"user_id":"user1","name":"Alice","country":"US"}
user2:{"user_id":"user2","name":"Bob","country":"UK"}
user3:{"user_id":"user3","name":"Charlie","country":"CA"}
```

In the ksqlDB CLI, create a table from profiles:
```sql
CREATE TABLE user_profiles (
  user_id VARCHAR PRIMARY KEY,
  name VARCHAR,
  country VARCHAR
) WITH (
  KAFKA_TOPIC='user-profiles',
  VALUE_FORMAT='JSON'
);
```

Join events with user profiles:
```sql
CREATE STREAM enriched_user_events AS
  SELECT e.user_id,
         e.event_type,
         e.timestamp,
         p.name,
         p.country
  FROM user_events_stream e
  LEFT JOIN user_profiles p ON e.user_id = p.user_id
  EMIT CHANGES;
```

Query the enriched events:
```sql
SELECT * FROM enriched_user_events EMIT CHANGES;
```

### Task 8: Inspect and Manage

List all streams:
```sql
SHOW STREAMS;
```

List all tables:
```sql
SHOW TABLES;
```

Describe a stream:
```sql
DESCRIBE user_events_stream;
```

Describe extended (shows query):
```sql
DESCRIBE EXTENDED enriched_user_events;
```

Show running queries:
```sql
SHOW QUERIES;
```

Terminate a query (use the query ID from SHOW QUERIES):
```sql
TERMINATE <query_id>;
```

## Verification

You've successfully completed this exercise when you can:
- ✅ Create streams from Kafka topics
- ✅ Query streams with filtering
- ✅ Create derived streams with transformations
- ✅ Create tables with aggregations
- ✅ Perform windowed aggregations
- ✅ Join streams and tables
- ✅ Manage and inspect ksqlDB objects

## Key Concepts

- **Stream**: Immutable, append-only sequence of events (unbounded)
- **Table**: Mutable collection representing current state (changelog)
- **Push Query**: Continuously emits results as data arrives (`EMIT CHANGES`)
- **Pull Query**: Returns current state snapshot (tables only)
- **Materialized View**: Continuously updated query result stored as a table
- **Window**: Time-based grouping for aggregations

## Common Use Cases

- **Real-time Analytics**: Count events, calculate metrics
- **Stream Enrichment**: Join events with reference data
- **Filtering**: Route specific events to different streams
- **Aggregations**: Session windows, tumbling windows, hopping windows
- **Alerting**: Detect patterns and trigger actions

## Cleanup

Stop the containers:
```bash
docker compose down
```

## Next Steps

Explore more advanced ksqlDB features:
- User-defined functions (UDFs)
- Pull queries for serving applications
- Schema Registry integration
- Complex event processing patterns

## Additional Resources

- [ksqlDB Documentation](https://docs.ksqldb.io/)
- [ksqlDB Tutorials](https://kafka-tutorials.confluent.io/)
- [ksqlDB Query Guide](https://docs.ksqldb.io/en/latest/developer-guide/ksqldb-reference/)
