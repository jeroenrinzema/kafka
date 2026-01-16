# Exercise 6.02: Benchmarking and Performance Tuning

## Learning Objectives

- Use Kafka's built-in performance testing tools
- Understand producer and consumer throughput metrics
- Visualize performance in Grafana dashboards
- Identify bottlenecks and tune Kafka configurations
- Apply performance optimizations for production workloads

## Background

Before deploying Kafka to production or making configuration changes, you need to understand your cluster's performance characteristics. Kafka provides built-in tools for benchmarking:

- **kafka-producer-perf-test.sh**: Measures producer throughput and latency
- **kafka-consumer-perf-test.sh**: Measures consumer throughput

Combined with Grafana monitoring, you can visualize the impact of different configurations in real-time.

### Key Performance Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| **Throughput** | Messages/second or MB/second | Maximize |
| **Latency** | Time from send to acknowledge | Minimize |
| **CPU Usage** | Broker CPU utilization | < 70% |
| **Disk I/O** | Read/write operations | Monitor for saturation |
| **Network** | Bytes in/out | Monitor for saturation |

### Factors Affecting Performance

1. **Batch size**: Larger batches = higher throughput. Very small batches cause high latency due to request overhead.
2. **Compression**: Reduces network I/O for compressible data, increases CPU usage. Random/binary data may not benefit.
3. **Replication factor**: More replicas = more network traffic
4. **Acknowledgments (acks)**: `acks=all` is slower but safer (difference is visible with replication-factor > 1)
5. **Partitions**: More partitions = more parallelism

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Performance Testing Stack                        │
│                                                                          │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐             │
│  │   Producer   │     │    Kafka     │     │   Consumer   │             │
│  │   Perf Test  │────▶│   Cluster    │────▶│   Perf Test  │             │
│  └──────────────┘     └──────────────┘     └──────────────┘             │
│                              │                                           │
│                              │ JMX Metrics                               │
│                              ▼                                           │
│                       ┌──────────────┐     ┌──────────────┐             │
│                       │ JMX Exporter │────▶│  Prometheus  │             │
│                       └──────────────┘     └──────────────┘             │
│                                                   │                      │
│                                                   ▼                      │
│                                            ┌──────────────┐             │
│                                            │   Grafana    │             │
│                                            │  Dashboard   │             │
│                                            └──────────────┘             │
└─────────────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Docker and Docker Compose installed
- Completed exercise 6.01 (Grafana Monitoring)

## Setup

Start the monitoring stack:

```bash
docker compose up -d
```

Wait for all services to be healthy:

```bash
docker compose ps
```

## Accessing Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | - |
| Kafka | localhost:9092 | - |

## Tasks

### Task 1: Create a Test Topic

Create a topic for benchmarking with multiple partitions:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic perf-test \
  --bootstrap-server localhost:9092 \
  --partitions 6 \
  --replication-factor 1
```

Verify the topic:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic perf-test \
  --bootstrap-server localhost:9092
```

### Task 2: Open the Grafana Dashboard

1. Open Grafana: http://localhost:3000
2. Login with admin / admin
3. Navigate to Dashboards → Kafka Performance
4. Set the time range to "Last 5 minutes"
5. Enable auto-refresh (every 5 seconds)

Keep this dashboard open while running the performance tests.

### Task 3: Baseline Producer Performance Test

Run a basic producer performance test:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092
```

Parameters:
- `--num-records`: Total messages to send
- `--record-size`: Message size in bytes
- `--throughput -1`: No throttling (maximum speed)

Note the output metrics:
- Records/second
- MB/second
- Average latency
- 50th, 95th, 99th percentile latencies

**Watch the Grafana dashboard** to see:
- Messages In Per Second spike
- Bytes In Per Second increase
- Request rate changes

### Task 4: Consumer Performance Test

Run a consumer performance test:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-consumer-perf-test.sh \
  --topic perf-test \
  --bootstrap-server localhost:9092 \
  --messages 100000
```

Note the output:
- MB/second consumed
- Messages/second consumed

### Task 5: Test Different Batch Sizes

Compare performance with different batch sizes:

**Small batch (1 KB):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 batch.size=1024
```

**Large batch (64 KB):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 batch.size=65536
```

**Very large batch (256 KB):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 batch.size=262144
```

Compare the results:

| Batch Size | Throughput (MB/s) | Avg Latency (ms) |
|------------|-------------------|------------------|
| 1 KB | | |
| 64 KB | | |
| 256 KB | | |

**Expected pattern:** Small batches (1 KB) will have LOW throughput and HIGH latency because each message creates a separate network request, causing congestion. Larger batches improve both throughput AND latency up to a point.

### Task 6: Test Compression

Compare different compression algorithms:

**No compression (baseline):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 compression.type=none
```

**GZIP compression:**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 compression.type=gzip
```

**LZ4 compression:**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 compression.type=lz4
```

**Snappy compression:**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 compression.type=snappy
```

**ZSTD compression:**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 compression.type=zstd
```

Compression comparison:

| Compression | Throughput | CPU Usage | Ratio |
|-------------|------------|-----------|-------|
| none | | | 1.0x |
| gzip | | | |
| lz4 | | | |
| snappy | | | |
| zstd | | | |

Watch the Grafana dashboard for CPU usage changes.

**Note:** The test uses random data which doesn't compress well. With random data, `compression.type=none` may be fastest. In production with compressible data (JSON, logs, text), compression can significantly improve throughput by reducing network I/O.

### Task 7: Test Acknowledgment Modes

Compare different acks settings:

**acks=0 (fire and forget):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 acks=0
```

**acks=1 (leader only):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 acks=1
```

**acks=all (all replicas):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 acks=all
```

Results:

| Acks | Throughput | Latency | Durability |
|------|------------|---------|------------|
| 0 | | | None |
| 1 | | | Leader |
| all | | | Full |

**Note:** In this exercise, the topic has `replication-factor=1`, so `acks=all` behaves similarly to `acks=1`. The significant performance difference between acks=1 and acks=all is only visible with replication-factor > 1, where the broker must wait for replicas to acknowledge.

### Task 8: Test Linger Time

Linger time allows batching more messages:

**No linger (default):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 linger.ms=0
```

**5ms linger:**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 linger.ms=5
```

**50ms linger:**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props bootstrap.servers=localhost:9092 linger.ms=50
```

### Task 9: High-Throughput Configuration Test

Combine optimizations for maximum throughput:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 500000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props \
    bootstrap.servers=localhost:9092 \
    batch.size=262144 \
    linger.ms=10 \
    compression.type=lz4 \
    acks=1 \
    buffer.memory=67108864
```

Watch Grafana to see peak throughput.

### Task 10: Low-Latency Configuration Test

Configure for minimum latency:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props \
    bootstrap.servers=localhost:9092 \
    batch.size=16384 \
    linger.ms=0 \
    compression.type=none \
    acks=1
```

**Note:** We use `batch.size=16384` (16 KB, the default) rather than `batch.size=1`. Setting batch.size=1 would force each message to be sent individually, creating massive overhead from thousands of tiny network requests, which paradoxically *increases* latency due to network congestion and request queuing. The key low-latency setting is `linger.ms=0`, which sends batches immediately without waiting to accumulate more messages.

Compare latency percentiles with the high-throughput test.

### Task 11: Consumer Performance with Different Fetch Sizes

Test consumer performance with different fetch configurations:

**Default fetch size:**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-consumer-perf-test.sh \
  --topic perf-test \
  --bootstrap-server localhost:9092 \
  --messages 500000
```

**Larger fetch size (1 MB):**

```bash
docker exec -it kafka /opt/kafka/bin/kafka-consumer-perf-test.sh \
  --topic perf-test \
  --bootstrap-server localhost:9092 \
  --messages 500000 \
  --fetch-size 1048576
```

**Note:** The `--threads` option has been deprecated in recent Kafka versions and is ignored. To achieve consumer parallelism in production, run multiple consumer instances in the same consumer group, with each consumer assigned to different partitions. The maximum parallelism equals the number of partitions in the topic.

### Task 12: Analyze Grafana Metrics

Review the performance dashboard and identify:

1. **Messages In Per Second**: Peak throughput achieved
2. **Request Latency**: Average and percentile latencies
3. **CPU Usage**: Broker CPU during tests
4. **Network I/O**: Bytes in/out per second
5. **Disk I/O**: Log flush rates

Create a screenshot or note the key metrics for each test configuration.

### Task 13: Create a Performance Report (Optional)

Fill in this performance comparison table:

| Configuration | Records/sec | MB/sec | Avg Latency | P99 Latency |
|---------------|-------------|--------|-------------|-------------|
| Baseline | | | | |
| Large batch (256KB) | | | | |
| LZ4 compression | | | | |
| acks=0 | | | | |
| High-throughput combo | | | | |
| Low-latency combo | | | | |

## Verification

You've successfully completed this exercise when:
- ✅ You can run producer and consumer performance tests
- ✅ You understand the impact of batch size on throughput
- ✅ You can compare compression algorithms
- ✅ You understand acks mode trade-offs
- ✅ You can visualize performance metrics in Grafana

## Performance Tuning Guidelines

### For High Throughput

```properties
# Producer
batch.size=262144          # 256 KB batches
linger.ms=10               # Wait up to 10ms to batch
compression.type=lz4       # Fast compression
acks=1                     # Leader acknowledgment only
buffer.memory=67108864     # 64 MB buffer

# Broker
num.io.threads=8
num.network.threads=8
socket.receive.buffer.bytes=102400
socket.send.buffer.bytes=102400
```

### For Low Latency

```properties
# Producer
batch.size=16384           # Default batch size (don't set too small!)
linger.ms=0                # Send immediately, don't wait to batch
compression.type=none      # No compression overhead
acks=1                     # Don't wait for all replicas

# Broker
num.io.threads=8
log.flush.interval.messages=1
```

**Important:** Avoid setting `batch.size` too small (e.g., 1). This creates excessive network overhead and actually *increases* latency. The key to low latency is `linger.ms=0`, which sends batches immediately.

### For Durability

```properties
# Producer
acks=all                   # All replicas must acknowledge
retries=3                  # Retry on failure
enable.idempotence=true    # Exactly-once semantics

# Broker
min.insync.replicas=2      # Require 2 replicas
unclean.leader.election.enable=false
```

## Key Takeaways

1. **Batching improves throughput** - very small batches hurt both throughput AND latency due to request overhead
2. **LZ4** offers good throughput/compression balance for compressible data; random data may not benefit from compression
3. **acks=all** provides durability; performance impact is visible with replication-factor > 1
4. **Partition count** limits consumer parallelism
5. **Monitor Grafana** to identify bottlenecks
6. **Test your specific workload** - results vary based on data, hardware, and configuration

## Troubleshooting

### Low throughput

- Check broker CPU and disk I/O in Grafana
- Increase batch.size and linger.ms
- Enable compression
- Add more partitions

### High latency

- Reduce batch.size
- Set linger.ms=0
- Use acks=1 instead of acks=all
- Check network latency

### Consumer can't keep up

- Add more partitions
- Increase consumer threads
- Check fetch.min.bytes and fetch.max.wait.ms

## Cleanup

Stop and remove all containers:

```bash
docker compose down
```

Remove all data:

```bash
docker compose down -v
```

## Additional Resources

- [Kafka Performance Tuning](https://kafka.apache.org/documentation/#producerconfigs)
- [Confluent Performance Testing](https://docs.confluent.io/kafka/operations-tools/kafka-tools.html#kafka-producer-perf-test-sh)
- [Optimizing Kafka Producers](https://developer.confluent.io/tutorials/optimize-producer-throughput/kafka.html)

## Next Steps

Continue to [Exercise 7.01: Partition Reassignment](../7.01-partition-reassignment/)