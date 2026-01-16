# Exercise 7.02: Client Quotas & Throttling

## Learning Objectives

- Understand Kafka's quota system for rate limiting
- Configure producer and consumer bandwidth quotas
- Set request rate quotas to protect brokers
- Apply quotas per user, client-id, or both
- Monitor throttled requests
- Handle "noisy neighbor" scenarios in multi-tenant clusters

## Background

In shared Kafka clusters, one misbehaving client can overwhelm the cluster and impact all other users. Quotas solve this by limiting:

1. **Producer bandwidth**: Bytes/second a producer can send
2. **Consumer bandwidth**: Bytes/second a consumer can fetch
3. **Request rate**: Number of requests per second

### When to Use Quotas

- **Multi-tenant environments**: Prevent one team from consuming all resources
- **Cost control**: Limit data transfer for budget management
- **Stability**: Protect cluster from sudden traffic spikes
- **Fair sharing**: Ensure all applications get reasonable throughput

### Quota Types

| Quota Type | Description | Default |
|------------|-------------|---------|
| `producer_byte_rate` | Max bytes/sec for producing | Unlimited |
| `consumer_byte_rate` | Max bytes/sec for consuming | Unlimited |
| `request_percentage` | % of I/O threads for request handling | Unlimited |

### Important: Per-Broker Quotas

**Quotas are enforced per-broker, not cluster-wide.** This means:

- A 1 MB/s quota allows 1 MB/s to *each* broker
- In a 3-broker cluster with data spread across all brokers, effective throughput ≈ 3 MB/s
- The quota limits the rate at which each broker will accept data from a client

This design ensures quotas scale with the cluster and prevents a single broker from becoming a bottleneck.

### Quota Entities

Quotas can be applied at different levels (most specific wins):

1. **User + Client-ID**: `user=alice, client-id=producer-1`
2. **User only**: `user=alice`
3. **Client-ID only**: `client-id=producer-1`
4. **Default user + Client-ID**: `user=<default>, client-id=producer-1`
5. **Default Client-ID**: `client-id=<default>`
6. **Default user**: `user=<default>`

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kafka Cluster                                │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Quota Manager                          │  │
│  │                                                           │  │
│  │   User Quotas     Client-ID Quotas    Default Quotas     │  │
│  │   ┌─────────┐     ┌─────────┐         ┌─────────┐        │  │
│  │   │ alice   │     │ app-1   │         │ default │        │  │
│  │   │ 10 MB/s │     │ 5 MB/s  │         │ 1 MB/s  │        │  │
│  │   └─────────┘     └─────────┘         └─────────┘        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                         │
│  │ Broker 1│  │ Broker 2│  │ Broker 3│                         │
│  └─────────┘  └─────────┘  └─────────┘                         │
└─────────────────────────────────────────────────────────────────┘
           ▲                    ▲                    ▲
           │                    │                    │
    ┌──────┴───────┐    ┌──────┴───────┐    ┌──────┴───────┐
    │ Producer     │    │ Producer     │    │ Consumer     │
    │ client-id:   │    │ client-id:   │    │ client-id:   │
    │ app-1        │    │ app-2        │    │ app-1        │
    │ (5 MB/s max) │    │ (unlimited)  │    │ (5 MB/s max) │
    └──────────────┘    └──────────────┘    └──────────────┘
```

## Prerequisites

- Completed exercises 1.01-1.06 (Kafka fundamentals)
- Docker and Docker Compose installed

## Setup

Start the Kafka cluster:

```bash
docker compose up -d
```

Wait about 15 seconds for the cluster to initialize.

Verify the cluster is running:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092
```

## Tasks

### Task 1: Create Test Topics

Create topics for testing:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic events \
  --bootstrap-server localhost:9092 \
  --partitions 6 \
  --replication-factor 2

docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic logs \
  --bootstrap-server localhost:9092 \
  --partitions 3 \
  --replication-factor 2
```

### Task 2: Baseline Test Without Quotas

First, let's see unrestricted performance:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 50000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 client.id=fast-producer
```

Note the throughput (MB/sec). This is the unrestricted baseline.

### Task 3: View Current Quotas

List all configured quotas:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --all
```

Initially, no quotas are configured.

### Task 4: Set a Client-ID Quota

Limit the `slow-producer` client to 1 MB/s:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'producer_byte_rate=1048576' \
  --entity-type clients \
  --entity-name slow-producer
```

> **Note**: 1048576 bytes = 1 MB

Verify the quota was set:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --entity-name slow-producer
```

### Task 5: Test the Throttled Producer

Run the producer with the throttled client-id:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 10000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 client.id=slow-producer
```

Compare the throughput to the baseline. You should see:

- **Reduced throughput**: Limited by the quota (approximately 1 MB/s per broker)
- **Increased latency**: The broker delays responses when the quota is exceeded
- **Throttle time**: Visible in the latency metrics

> **Note**: Since quotas are per-broker and data is distributed across multiple brokers, the observed throughput may be higher than 1 MB/s. With 3 brokers and 6 partitions, you might see ~3 MB/s effective throughput.

To see stronger throttling effect, use smaller batches:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 10000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 client.id=slow-producer batch.size=1000 linger.ms=0
```

With smaller batches, you'll see much higher latencies (seconds) as the broker throttles each request.

### Task 6: Set Consumer Quotas

Limit consumer bandwidth for `slow-consumer`:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'consumer_byte_rate=524288' \
  --entity-type clients \
  --entity-name slow-consumer
```

This limits consumption to 512 KB/s.

Test it:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-consumer-perf-test.sh \
  --broker-list localhost:9092 \
  --topic events \
  --messages 20000 \
  --print-metrics \
  -- --group slow-group --consumer-property client.id=slow-consumer
```

The fetch throughput should be limited to approximately 512 KB/s.

### Task 7: Set Default Quotas

Set a default quota for all clients without a specific quota:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'producer_byte_rate=5242880,consumer_byte_rate=5242880' \
  --entity-type clients \
  --entity-default
```

This sets 5 MB/s as the default for both producing and consuming.

Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --entity-default
```

### Task 8: Test Default Quota

Test with a new client-id that has no specific quota:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 20000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 client.id=new-producer
```

The throughput should be limited to approximately 5 MB/s (the default).

### Task 9: User-Based Quotas

For SASL-authenticated clusters, you can set quotas per user. Let's simulate this:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'producer_byte_rate=10485760,consumer_byte_rate=20971520' \
  --entity-type users \
  --entity-name alice
```

This gives user `alice`:
- 10 MB/s producer quota
- 20 MB/s consumer quota

View user quotas:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type users \
  --entity-name alice
```

### Task 10: Combined User + Client-ID Quotas

Set a quota for a specific user AND client-id combination:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'producer_byte_rate=2097152' \
  --entity-type users \
  --entity-name alice \
  --entity-type clients \
  --entity-name batch-producer
```

This limits user `alice` with client-id `batch-producer` to 2 MB/s.

### Task 11: Request Rate Quotas

Limit the percentage of broker I/O threads a client can use:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'request_percentage=25' \
  --entity-type clients \
  --entity-name limited-client
```

This limits `limited-client` to 25% of broker request handling capacity.

### Task 12: Monitor Throttled Requests

Produce rapidly to trigger throttling:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 50000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 client.id=slow-producer &
```

While it's running, check throttle metrics via JMX (if configured) or observe the reduced throughput in the output.

### Task 13: List All Quotas

View all configured quotas in the cluster:

```bash
echo "=== Client Quotas ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --all

echo ""
echo "=== User Quotas ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type users \
  --all

echo ""
echo "=== Default Client Quota ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --entity-default

echo ""
echo "=== Default User Quota ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type users \
  --entity-default
```

### Task 14: Modify an Existing Quota

Increase the quota for `slow-producer`:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'producer_byte_rate=2097152' \
  --entity-type clients \
  --entity-name slow-producer
```

Quotas take effect immediately - no restart needed!

### Task 15: Remove a Quota

Remove the quota for a specific client:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --delete-config 'producer_byte_rate' \
  --entity-type clients \
  --entity-name slow-producer
```

Verify it's gone:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --entity-name slow-producer
```

### Task 16: Simulate "Noisy Neighbor" Scenario

Let's see how quotas protect against a misbehaving client.

First, ensure the default quota is set:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'producer_byte_rate=1048576' \
  --entity-type clients \
  --entity-name noisy-producer
```

Start a "well-behaved" producer in the background:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 100000 \
  --record-size 1000 \
  --throughput 500 \
  --producer-props bootstrap.servers=localhost:9092 client.id=good-producer &
```

Start the "noisy" producer that tries to overwhelm the cluster:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic events \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 client.id=noisy-producer
```

The noisy producer will be throttled to 1 MB/s while the good producer continues unaffected.

### Task 17: View Quota Metrics in Kafka UI

Open http://localhost:8080 and:

1. Navigate to Brokers
2. Look at the metrics section
3. Observe throttling indicators (if available in your UI version)

### Task 18: Export and Import Quota Configuration

Export current quotas to a file (useful for backup/migration):

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --all > quotas-backup.txt

cat quotas-backup.txt
```

## Key Concepts

### How Throttling Works

1. Broker tracks bytes sent/received per client
2. When quota exceeded, broker calculates delay time
3. Response is delayed, causing client to slow down
4. Client sees increased latency, not errors

### Quota Calculation Window

- Quotas are measured in a sliding window (default: 1 second)
- Configurable via `quota.window.num` and `quota.window.size.seconds`

### Quota Precedence

Most specific quota wins:

```
user + client-id  >  user  >  client-id  >  default user+client  >  default
```

### Dynamic Updates

- Quotas are stored in ZooKeeper/KRaft metadata
- Changes take effect immediately
- No broker restart required

## Troubleshooting

### Producer Very Slow

Check if it's being throttled:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --entity-name <client-id>
```

### Quota Not Taking Effect

- Verify the client-id matches exactly (case-sensitive)
- Check for more specific quotas that might override
- Ensure the client is actually using the expected client-id

### Quotas Not Visible

```bash
# List ALL quotas including defaults
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type clients \
  --all
```

## Cleanup

Stop all services:

```bash
docker compose down -v
```

## Best Practices

1. **Start with conservative defaults**: Set a default quota that protects the cluster
2. **Grant exceptions explicitly**: Increase quotas for known, trusted applications
3. **Monitor throttle metrics**: Alert when clients are consistently throttled
4. **Use client-id naming conventions**: Make it easy to identify and manage quotas
5. **Document quota policies**: Maintain a registry of quotas and their justifications
6. **Test quota changes**: Verify in non-production before applying
7. **Combine with ACLs**: Use quotas alongside access controls for defense in depth
8. **Review periodically**: Quotas may need adjustment as usage patterns change

## Common Quota Configurations

### Conservative Multi-Tenant Setup

```bash
# Default for all clients
--add-config 'producer_byte_rate=1048576,consumer_byte_rate=5242880'
--entity-type clients --entity-default

# Trusted application with higher limits
--add-config 'producer_byte_rate=52428800,consumer_byte_rate=104857600'
--entity-type clients --entity-name trusted-app
```

### Development Environment

```bash
# Limit all development to prevent runaway tests
--add-config 'producer_byte_rate=10485760,consumer_byte_rate=20971520'
--entity-type clients --entity-default
```

### Per-Team Quotas (with SASL)

```bash
# Team A - high priority
--add-config 'producer_byte_rate=104857600'
--entity-type users --entity-name team-a-service

# Team B - lower priority
--add-config 'producer_byte_rate=20971520'
--entity-type users --entity-name team-b-service
```

## Additional Resources

- [Kafka Quotas Documentation](https://kafka.apache.org/documentation/#design_quotas)
- [KIP-124: Request Rate Quotas](https://cwiki.apache.org/confluence/display/KAFKA/KIP-124+-+Request+rate+quotas)
- [Confluent Quotas Guide](https://docs.confluent.io/platform/current/kafka/design.html#quotas)

## Next Steps

Try these challenges:

1. Set up quotas that vary by time of day (requires external automation)
2. Create a script that automatically adjusts quotas based on cluster load
3. Implement quota monitoring with Prometheus and Grafana
4. Design a quota policy for a 10-team organization sharing one cluster
5. Test how quotas interact with producer batching and compression