# Exercise 3.04: Change Data Capture with Debezium

## Learning Objectives

- Understand Change Data Capture (CDC) concepts
- Configure Debezium PostgreSQL connector
- Capture INSERT, UPDATE, and DELETE events as Kafka messages
- Understand Debezium event structure (before/after states)
- Handle database schema changes
- Configure topic routing and naming

## Background

### What is Change Data Capture (CDC)?

CDC is a pattern that tracks changes in a database and streams them to other systems. Instead of querying the database periodically (polling), CDC captures changes as they happen by reading the database's transaction log.

### Why Debezium?

Debezium is the industry-standard open-source CDC platform. It:
- Reads database transaction logs (not polling)
- Captures all changes, including DELETEs
- Provides before/after snapshots for UPDATEs
- Guarantees ordering per row
- Is fully open source (Apache 2.0 license)

### Supported Databases

| Database | Transaction Log |
|----------|----------------|
| PostgreSQL | Write-Ahead Log (WAL) |
| MySQL | Binary Log (binlog) |
| MongoDB | Oplog |
| SQL Server | Transaction Log |
| Oracle | LogMiner / Xstream |

### CDC Event Types

| Operation | Description | Has "before"? | Has "after"? |
|-----------|-------------|---------------|--------------|
| `c` (create) | INSERT | No | Yes |
| `u` (update) | UPDATE | Yes | Yes |
| `d` (delete) | DELETE | Yes | No |
| `r` (read) | Initial snapshot | No | Yes |

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│  ┌─────────────┐     ┌─────────────────┐     ┌─────────────────────┐   │
│  │ PostgreSQL  │────►│    Debezium     │────►│   Kafka Topics      │   │
│  │  Database   │ WAL │   Connector     │     │                     │   │
│  │             │     │                 │     │ ┌─────────────────┐ │   │
│  │ ┌─────────┐ │     │  Reads WAL and  │     │ │ dbserver.shop.  │ │   │
│  │ │customers│ │     │  converts to    │     │ │   customers     │ │   │
│  │ ├─────────┤ │     │  Kafka events   │     │ ├─────────────────┤ │   │
│  │ │products │ │     │                 │     │ │ dbserver.shop.  │ │   │
│  │ ├─────────┤ │     │                 │     │ │   products      │ │   │
│  │ │orders   │ │     │                 │     │ ├─────────────────┤ │   │
│  │ ├─────────┤ │     │                 │     │ │ dbserver.shop.  │ │   │
│  │ │order_   │ │     │                 │     │ │   orders        │ │   │
│  │ │items    │ │     │                 │     │ └─────────────────┘ │   │
│  │ └─────────┘ │     │                 │     │                     │   │
│  └─────────────┘     └─────────────────┘     └─────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Docker and Docker Compose installed
- Completed Exercise 3.03 (Kafka Connect Fundamentals)
- Basic SQL knowledge

## Setup

Start the environment:

```bash
docker compose up -d
```

This starts:
- **Kafka** - Message broker
- **PostgreSQL** - Source database with sample e-commerce data
- **Debezium Connect** - Kafka Connect with Debezium connectors
- **Kafka UI** - Web interface for monitoring

Wait about 30 seconds for all services to start.

**Verify PostgreSQL is ready:**

```bash
docker exec postgres psql -U postgres -d ecommerce -c "SELECT count(*) FROM shop.customers;"
```

You should see 5 customers.

> **Note**: PostgreSQL is exposed on port 5433 (not the default 5432) to avoid conflicts with local PostgreSQL installations. The internal port is still 5432, which is what the containers use.

**Verify Debezium Connect is ready:**

```bash
curl -s http://localhost:8083/connector-plugins | jq '.[].class' | grep -i debezium
```

You should see `io.debezium.connector.postgresql.PostgresConnector`.

## Tasks

### Task 1: Explore the Sample Database

Connect to PostgreSQL and explore the schema:

```bash
docker exec postgres psql -U postgres -d ecommerce
```

**List tables:**

```sql
\dt shop.*
```

**View table structures:**

```sql
\d shop.customers
\d shop.orders
```

**Check sample data:**

```sql
SELECT * FROM shop.customers;
SELECT * FROM shop.products LIMIT 5;
SELECT * FROM shop.orders;
```

**Exit psql:**

```sql
\q
```

### Task 2: Create the Debezium PostgreSQL Connector

Create a connector to capture changes from all tables in the `shop` schema:

```bash
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ecommerce-connector",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "postgres",
      "database.port": "5432",
      "database.user": "postgres",
      "database.password": "postgres",
      "database.dbname": "ecommerce",
      "topic.prefix": "dbserver",
      "schema.include.list": "shop",
      "plugin.name": "pgoutput",
      "publication.name": "debezium_publication",
      "slot.name": "debezium_slot",
      "decimal.handling.mode": "string",
      "time.precision.mode": "connect"
    }
  }' | jq
```

**Key configuration options:**

| Option | Description |
|--------|-------------|
| `topic.prefix` | Prefix for all generated Kafka topics |
| `schema.include.list` | Which database schemas to capture |
| `plugin.name` | PostgreSQL logical decoding plugin (pgoutput is built-in) |
| `publication.name` | PostgreSQL publication to use |
| `slot.name` | Replication slot name (stores position in WAL) |

**Verify the connector is running:**

```bash
curl -s http://localhost:8083/connectors/ecommerce-connector/status | jq
```

Wait for the status to show `RUNNING` for both the connector and task.

### Task 3: Observe Initial Snapshot

When a connector starts, Debezium performs an initial snapshot of existing data. Check the topics created:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list | grep dbserver
```

You should see topics like:
- `dbserver.shop.customers`
- `dbserver.shop.products`
- `dbserver.shop.orders`
- `dbserver.shop.order_items`

**Consume the initial snapshot from customers:**

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dbserver.shop.customers \
  --from-beginning \
  --max-messages 5
```

### Task 4: Understand the Event Structure

Let's look at a single event in detail:

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dbserver.shop.customers \
  --from-beginning \
  --max-messages 1 2>/dev/null | jq '.payload'
```

> **Note**: The full message includes both `schema` and `payload` fields. We use `.payload` to extract just the data. The schema describes the structure of the event and is useful for deserializers.

**Event structure (payload only):**

```json
{
  "before": null,
  "after": {
    "customer_id": 1,
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Johnson",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T10:00:00Z"
  },
  "source": {
    "version": "2.5.0.Final",
    "connector": "postgresql",
    "name": "dbserver",
    "ts_ms": 1705312800000,
    "snapshot": "first",
    "db": "ecommerce",
    "schema": "shop",
    "table": "customers",
    "txId": 1234,
    "lsn": 12345678
  },
  "op": "r",
  "ts_ms": 1705312800000
}
```

**Field descriptions:**

| Field | Description |
|-------|-------------|
| `before` | Row state before the change (null for INSERT/snapshot) |
| `after` | Row state after the change (null for DELETE) |
| `source` | Metadata about where the change came from |
| `op` | Operation type: `r`=read(snapshot), `c`=create, `u`=update, `d`=delete |
| `ts_ms` | Timestamp when Debezium processed the event |

### Task 5: Capture an INSERT Event

Open a new terminal to watch for changes:

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dbserver.shop.customers
```

In another terminal, insert a new customer:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  INSERT INTO shop.customers (email, first_name, last_name)
  VALUES ('frank@example.com', 'Frank', 'Miller');
"
```

**Observe the event in the consumer terminal:**

The event should have:
- `"op": "c"` (create)
- `"before": null`
- `"after": { ... }` containing the new customer data

### Task 6: Capture an UPDATE Event

Update a customer's email:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  UPDATE shop.customers
  SET email = 'alice.johnson@example.com'
  WHERE customer_id = 1;
"
```

**Observe the event:**

The event should have:
- `"op": "u"` (update)
- `"after": { ... }` with the new email

> **Note on `before` field**: By default, PostgreSQL only includes the primary key in the `before` field for UPDATE/DELETE events. To get full before-images, you need to set `REPLICA IDENTITY FULL` on the table:
> ```sql
> ALTER TABLE shop.customers REPLICA IDENTITY FULL;
> ```
> This increases WAL size but provides complete before/after snapshots.

When `before` is available, this structure is powerful for:
- Auditing what changed
- Building materialized views
- Synchronizing downstream systems

### Task 7: Capture a DELETE Event

Delete a customer:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  DELETE FROM shop.customers WHERE customer_id = 6;
"
```

**Observe the event:**

The event should have:
- `"op": "d"` (delete)
- `"before": { ... }` with at least the primary key (full row if REPLICA IDENTITY FULL is set)
- `"after": null`

> **Note**: After the delete event, Debezium also emits a "tombstone" message (null payload) for the same key. This is used by Kafka log compaction to remove the record.

### Task 8: Observe Multi-Table Transactions

Start watching the orders topic:

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dbserver.shop.orders &

docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dbserver.shop.order_items &
```

Create a new order with items (in a transaction):

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  BEGIN;
  
  INSERT INTO shop.orders (customer_id, status, total_amount, shipping_address)
  VALUES (2, 'PENDING', 179.98, '456 Oak Ave, Los Angeles, CA 90001')
  RETURNING order_id;
  
  INSERT INTO shop.order_items (order_id, product_id, quantity, unit_price)
  VALUES 
    (5, 2, 2, 29.99),
    (5, 7, 3, 39.99);
  
  COMMIT;
"
```

You should see:
1. One event in `dbserver.shop.orders`
2. Two events in `dbserver.shop.order_items`

All events from the same transaction share the same `txId` in the `source` field.

### Task 9: Handle Schema Changes

Debezium can handle schema evolution. Let's add a column:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  ALTER TABLE shop.customers ADD COLUMN phone VARCHAR(20);
"
```

Now update a customer with the new field:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  UPDATE shop.customers 
  SET phone = '+1-555-0123' 
  WHERE customer_id = 1;
"
```

**Observe the event:**

The `after` field should now include the `phone` column!

Debezium automatically detects schema changes and updates the event structure.

### Task 10: Check Message Keys

Debezium uses the primary key as the Kafka message key. This ensures:
- Messages for the same row go to the same partition
- Ordering is preserved per row

**View messages with keys:**

```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dbserver.shop.customers \
  --from-beginning \
  --property print.key=true \
  --property key.separator=" | " \
  --max-messages 3
```

The key contains the primary key value (e.g., `{"customer_id": 1}`).

### Task 11: Monitor Connector Metrics

Check the connector status:

```bash
curl -s http://localhost:8083/connectors/ecommerce-connector/status | jq
```

Check the replication slot in PostgreSQL:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  SELECT slot_name, plugin, slot_type, active 
  FROM pg_replication_slots;
"
```

Check the current LSN (Log Sequence Number) position:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  SELECT pg_current_wal_lsn();
"
```

### Task 12: View in Kafka UI

Open http://localhost:8080 in your browser.

1. Navigate to **Topics** in the left menu
2. Find the `dbserver.shop.*` topics
3. Click on a topic to see messages
4. Observe the message structure and keys

Navigate to **Kafka Connect**:
1. Click on the `debezium` connect cluster
2. View the `ecommerce-connector` status
3. Check the connector configuration

### Task 13: Configure Topic Routing (Optional)

You can customize how Debezium names topics. Update the connector:

```bash
curl -X PUT http://localhost:8083/connectors/ecommerce-connector/config \
  -H "Content-Type: application/json" \
  -d '{
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres",
    "database.port": "5432",
    "database.user": "postgres",
    "database.password": "postgres",
    "database.dbname": "ecommerce",
    "topic.prefix": "cdc",
    "schema.include.list": "shop",
    "plugin.name": "pgoutput",
    "publication.name": "debezium_publication",
    "slot.name": "debezium_slot",
    "decimal.handling.mode": "string",
    "time.precision.mode": "connect",
    "transforms": "route",
    "transforms.route.type": "org.apache.kafka.connect.transforms.RegexRouter",
    "transforms.route.regex": "cdc\\.shop\\.(.*)",
    "transforms.route.replacement": "ecommerce-$1-events"
  }' | jq
```

This routes topics like:
- `cdc.shop.customers` → `ecommerce-customers-events`

### Task 14: Filter Tables (Optional)

To capture only specific tables, use `table.include.list`:

```bash
curl -X PUT http://localhost:8083/connectors/ecommerce-connector/config \
  -H "Content-Type: application/json" \
  -d '{
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres",
    "database.port": "5432",
    "database.user": "postgres",
    "database.password": "postgres",
    "database.dbname": "ecommerce",
    "topic.prefix": "dbserver",
    "table.include.list": "shop.customers,shop.orders",
    "plugin.name": "pgoutput",
    "publication.name": "debezium_publication",
    "slot.name": "debezium_slot",
    "decimal.handling.mode": "string",
    "time.precision.mode": "connect"
  }' | jq
```

This captures only `customers` and `orders` tables, ignoring `products` and `order_items`.

## Verification

You've successfully completed this exercise when you can:

- ✅ Configure a Debezium PostgreSQL connector
- ✅ Capture initial snapshot data
- ✅ Observe INSERT events with `after` state
- ✅ Observe UPDATE events with `before` and `after` states
- ✅ Observe DELETE events with `before` state
- ✅ Handle schema changes (adding columns)
- ✅ Understand message keys and ordering
- ✅ Monitor connector status

## Key Concepts

### Exactly-Once Semantics

Debezium + Kafka provides at-least-once delivery. For exactly-once:
1. Use idempotent consumers
2. Store offsets with processed data in same transaction
3. Use Kafka transactions on the producer side

### Snapshot Modes

| Mode | Description |
|------|-------------|
| `initial` | Snapshot on first start, then stream changes (default) |
| `initial_only` | Only snapshot, don't stream changes |
| `never` | Never snapshot, only stream changes from current position |
| `when_needed` | Snapshot if no offset exists |

Configure with: `"snapshot.mode": "initial"`

### Handling Deletes

For compacted topics, Debezium can emit a "tombstone" (null value) after deletes:

```json
{
  "delete.handling.mode": "rewrite",
  "tombstones.on.delete": "true"
}
```

### Performance Tuning

| Setting | Default | Description |
|---------|---------|-------------|
| `max.batch.size` | 2048 | Max events per batch |
| `poll.interval.ms` | 500 | How often to poll for changes |
| `max.queue.size` | 8192 | Max events in internal queue |

## Troubleshooting

### Connector fails to start

**Check logs:**

```bash
docker logs connect 2>&1 | tail -50
```

**Common issues:**
- PostgreSQL not configured for logical replication
- Publication doesn't exist
- Replication slot already in use

### Replication slot issues

**List slots:**

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  SELECT * FROM pg_replication_slots;
"
```

**Drop a stuck slot:**

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  SELECT pg_drop_replication_slot('debezium_slot');
"
```

### Missing changes

1. Verify the table is in the publication
2. Check the connector is in RUNNING state
3. Verify `wal_level` is set to `logical`
4. Check for errors in Connect logs

### High WAL disk usage

If not consuming changes fast enough, WAL files accumulate:

```bash
docker exec postgres psql -U postgres -d ecommerce -c "
  SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn) AS lag_bytes
  FROM pg_replication_slots WHERE slot_name = 'debezium_slot';
"
```

## Cleanup

Stop all services:

```bash
docker compose down -v
```

## Best Practices

1. **Use dedicated database user** - Create a user with only replication permissions
2. **Monitor replication lag** - Alert if lag grows too large
3. **Set slot retention** - Prevent WAL files from filling disk
4. **Use schema registry** - For production, use Avro with schema registry
5. **Handle schema changes carefully** - Test DDL changes in non-prod first
6. **Secure credentials** - Use secrets management, not plaintext passwords
7. **Plan for snapshots** - Initial snapshots can be resource-intensive

## Additional Resources

- [Debezium Documentation](https://debezium.io/documentation/)
- [Debezium PostgreSQL Connector](https://debezium.io/documentation/reference/connectors/postgresql.html)
- [PostgreSQL Logical Replication](https://www.postgresql.org/docs/current/logical-replication.html)
- [Debezium Blog](https://debezium.io/blog/)

## Next Steps

Continue to [Exercise 3.05: Vector Integration](../3.05-vector-integration/) to learn how to collect logs with Vector and send them to Kafka.