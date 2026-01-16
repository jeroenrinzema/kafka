# Exercise 7.06: Dynamic Broker Configuration

## Learning Objectives

- Understand Kafka's dynamic configuration system
- Learn the difference between static, per-broker, and cluster-wide configs
- Practice hot-reloading configurations without broker restarts
- Master common performance tunables
- Implement safe configuration change and rollback procedures

## Background

### Static vs Dynamic Configuration

Kafka supports three types of configuration:

| Type | Location | Requires Restart | Scope |
|------|----------|------------------|-------|
| **Static (read-only)** | server.properties | Yes | Per-broker |
| **Per-broker dynamic** | Stored in Kafka | No | Single broker |
| **Cluster-wide dynamic** | Stored in Kafka | No | All brokers |

### Configuration Precedence

When the same setting exists at multiple levels:

```
Per-broker dynamic  >  Cluster-wide dynamic  >  Static (server.properties)
         ▲                     ▲                        ▲
     Highest               Middle                   Lowest
     Priority              Priority                 Priority
```

This means you can:
1. Set a cluster-wide default
2. Override it for specific brokers that need different values

### Why Dynamic Configuration?

- **No downtime**: Changes apply immediately without restarts
- **Gradual rollout**: Test on one broker before applying cluster-wide
- **Quick rollback**: Revert changes instantly if issues occur
- **Operational flexibility**: Tune performance in response to load patterns

### Configuration Categories

Not all configs are dynamically updatable. Kafka classifies configs as:

| Category | Dynamic? | Examples |
|----------|----------|----------|
| **read-only** | No | `broker.id`, `log.dirs`, `listeners` |
| **per-broker** | Yes | `num.network.threads`, `num.io.threads` |
| **cluster-wide** | Yes | `log.retention.ms`, `log.flush.interval.ms` |

> **Important**: Some configs have both static and dynamic variants. For example, `log.retention.hours` is static (requires restart), but `log.retention.ms` is dynamic (hot-reloadable). Always check documentation or test in non-production first.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Configuration Storage                         │
│                                                                  │
│   ┌──────────────────┐    ┌──────────────────────────────────┐  │
│   │ server.properties│    │    Kafka Metadata (KRaft)        │  │
│   │                  │    │                                  │  │
│   │  Static configs  │    │  ┌─────────────────────────────┐ │  │
│   │  (read-only)     │    │  │ Cluster-wide dynamic configs│ │  │
│   │                  │    │  └─────────────────────────────┘ │  │
│   │  - broker.id     │    │  ┌─────────────────────────────┐ │  │
│   │  - log.dirs      │    │  │ Per-broker dynamic configs  │ │  │
│   │  - listeners     │    │  │ (broker 1, 2, 3, ...)       │ │  │
│   └──────────────────┘    │  └─────────────────────────────┘ │  │
│                           └──────────────────────────────────┘  │
│                                                                  │
│   kafka-configs.sh reads/writes dynamic configs via Admin API   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Completed exercises 1.01-1.06 (Kafka fundamentals)
- Docker and Docker Compose installed

## Setup

Start the 3-broker cluster:

```bash
docker compose up -d
```

Wait for the cluster to initialize:

```bash
sleep 20
docker exec kafka-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server kafka-1:19092 2>&1 | grep -E "^kafka-[0-9]"
```

---

## Part 1: Viewing Current Configuration

### Task 1: View All Broker Configs

List all configuration for broker 1:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all
```

This shows a lot of output! Each line shows:
- Config name
- Current value
- Whether it's sensitive (passwords, etc.)
- Config source (STATIC_BROKER_CONFIG, DEFAULT_CONFIG, etc.)

### Task 2: View Only Non-Default Configs

Filter to see only configs that differ from defaults:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe
```

Without `--all`, you only see configs that have been explicitly set.

### Task 3: View Cluster-Wide Dynamic Configs

View configs set at the cluster level (applies to all brokers):

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --describe
```

Initially this should be empty (no cluster-wide overrides set).

### Task 4: Check Specific Config Values

Look at specific configs we'll be modifying:

```bash
echo "=== Current Thread Configuration ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep -E "num.network.threads|num.io.threads|log.cleaner.threads"

echo ""
echo "=== Current Retention Configuration ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep -E "log.retention"
```

---

## Part 2: Cluster-Wide Configuration Changes

Cluster-wide configs apply to all brokers and are the recommended approach for most settings.

### Task 5: Set Cluster-Wide Log Retention

Change log retention from 7 days to 3 days cluster-wide.

> **Note**: `log.retention.hours` is NOT dynamically updatable - you must use `log.retention.ms` instead.

```bash
# 3 days = 259200000 ms
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --add-config log.retention.ms=259200000
```

### Task 6: Verify the Change

Check that the cluster-wide config is set:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --describe
```

Verify it's reflected on individual brokers:

```bash
echo "=== Broker 1 ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep log.retention.ms

echo "=== Broker 2 ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 2 \
  --describe --all 2>/dev/null | grep log.retention.ms

echo "=== Broker 3 ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 3 \
  --describe --all 2>/dev/null | grep log.retention.ms
```

All brokers should now show `log.retention.ms=259200000`.

### Task 7: Set Multiple Configs at Once

You can set multiple configurations in a single command:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --add-config log.flush.interval.messages=10000,log.flush.interval.ms=1000
```

This sets:
- `log.flush.interval.messages=10000` (flush after 10000 messages)
- `log.flush.interval.ms=1000` (flush every 1 second)

Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --describe
```

---

## Part 3: Per-Broker Configuration

Per-broker configs override cluster-wide settings for specific brokers.

### Task 8: Set Per-Broker Thread Count

Suppose Broker 1 handles more traffic and needs more I/O threads:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --alter \
  --add-config num.io.threads=16
```

### Task 9: Verify Per-Broker Override

Check that only Broker 1 has the override:

```bash
echo "=== Broker 1 (should show 16) ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep num.io.threads

echo ""
echo "=== Broker 2 (should show 8 - default) ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 2 \
  --describe --all 2>/dev/null | grep num.io.threads

echo ""
echo "=== Broker 3 (should show 8 - default) ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 3 \
  --describe --all 2>/dev/null | grep num.io.threads
```

### Task 10: View Per-Broker Overrides Only

See what's specifically set on Broker 1:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe
```

This shows only the dynamic overrides, not defaults.

### Task 11: Set Network Thread Configuration

Increase network threads on Broker 1:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --alter \
  --add-config num.network.threads=6
```

Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe
```

---

## Part 4: Configuration Rollback

### Task 12: Remove a Per-Broker Override

Remove the `num.io.threads` override from Broker 1 (reverts to cluster-wide or default):

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --alter \
  --delete-config num.io.threads
```

Verify it's back to default:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep num.io.threads
```

### Task 13: Remove Cluster-Wide Override

Remove the cluster-wide log retention setting:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --delete-config log.retention.ms
```

Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --describe
```

### Task 14: Remove Multiple Configs

Remove multiple configs at once:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --delete-config log.flush.interval.messages,log.flush.interval.ms
```

---

## Part 5: Topic-Level Configuration

Topics can also have dynamic configurations that override broker defaults.

### Task 15: Create a Test Topic

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic events \
  --bootstrap-server kafka-1:19092 \
  --partitions 3 \
  --replication-factor 3
```

### Task 16: View Topic Configuration

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type topics \
  --entity-name events \
  --describe --all
```

### Task 17: Set Topic-Specific Retention

Set a shorter retention period for this specific topic:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type topics \
  --entity-name events \
  --alter \
  --add-config retention.ms=86400000
```

This sets retention to 1 day (86400000 ms) for the `events` topic only.

### Task 18: Verify Topic Configuration

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type topics \
  --entity-name events \
  --describe
```

### Task 19: Set Multiple Topic Configs

Configure the topic for high-throughput:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type topics \
  --entity-name events \
  --alter \
  --add-config segment.bytes=268435456,cleanup.policy=delete,max.message.bytes=1048576
```

Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type topics \
  --entity-name events \
  --describe
```

---

## Part 6: Common Performance Tunables

### Task 20: Document Current Performance Settings

Create a baseline of current performance-related configs:

```bash
echo "=== Thread Configuration ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep -E "num\.(network|io|replica|recovery)\.threads"

echo ""
echo "=== Network Configuration ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep -E "socket\.(send|receive)\.buffer|replica\.socket"

echo ""
echo "=== Log Configuration ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep -E "log\.(flush|retention|segment)"
```

### Task 21: Apply High-Throughput Configuration

For a high-throughput workload, set cluster-wide performance configs:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --add-config num.io.threads=16,num.network.threads=6,num.replica.fetchers=4
```

### Task 22: Verify Applied Configuration

```bash
echo "=== Cluster-Wide Overrides ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --describe

echo ""
echo "=== Effective on Broker 1 ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep -E "num\.(io|network|replica)\.threads"
```

### Task 23: Rollback Performance Configuration

Remove the performance overrides:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --delete-config num.io.threads,num.network.threads,num.replica.fetchers
```

---

## Part 7: Safe Configuration Change Procedure

### Task 24: Implement a Safe Change Process

Follow this procedure for production config changes:

**Step 1: Document current state**

```bash
echo "=== Before Change - $(date) ===" > config-backup.txt
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --describe >> config-backup.txt

cat config-backup.txt
```

**Step 2: Apply to one broker first (canary)**

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --alter \
  --add-config log.cleaner.threads=2
```

**Step 3: Monitor and validate (in production, wait and watch metrics)**

```bash
echo "=== Broker 1 Config ==="
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe

# In production, monitor:
# - Broker CPU/memory usage
# - Request latency
# - Error rates
# - Log cleaner lag
```

**Step 4: If successful, apply cluster-wide**

```bash
# Remove per-broker override
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --alter \
  --delete-config log.cleaner.threads

# Apply cluster-wide
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --add-config log.cleaner.threads=2
```

**Step 5: Verify all brokers**

```bash
for broker in 1 2 3; do
  echo "=== Broker $broker ==="
  docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
    --bootstrap-server kafka-1:19092 \
    --entity-type brokers \
    --entity-name $broker \
    --describe --all 2>/dev/null | grep log.cleaner.threads
done
```

### Task 25: Practice Rollback

Rollback the change:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --delete-config log.cleaner.threads
```

Verify rollback:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --describe
```

---

## Common Dynamic Configurations Reference

### Thread Configuration

| Config | Default | Description |
|--------|---------|-------------|
| `num.network.threads` | 3 | Threads handling network requests |
| `num.io.threads` | 8 | Threads for disk I/O |
| `num.replica.fetchers` | 1 | Threads for replicating from leader |
| `num.recovery.threads.per.data.dir` | 1 | Threads for log recovery/flushing |
| `log.cleaner.threads` | 1 | Threads for log compaction |
| `background.threads` | 10 | Threads for background tasks |

### Log Retention Configuration

| Config | Default | Dynamic? | Description |
|--------|---------|----------|-------------|
| `log.retention.hours` | 168 (7 days) | **No** | Time-based retention (static) |
| `log.retention.ms` | - | **Yes** | Retention in milliseconds (use this for dynamic changes) |
| `log.retention.bytes` | -1 (unlimited) | **No** | Size-based retention per partition |
| `log.segment.bytes` | 1073741824 (1 GB) | **No** | Size of each log segment |
| `log.flush.interval.messages` | 9223372036854775807 | **Yes** | Messages before flush |
| `log.flush.interval.ms` | - | **Yes** | Time before flush |

### Network Configuration

| Config | Default | Description |
|--------|---------|-------------|
| `socket.send.buffer.bytes` | 102400 | SO_SNDBUF size |
| `socket.receive.buffer.bytes` | 102400 | SO_RCVBUF size |
| `socket.request.max.bytes` | 104857600 | Max request size |
| `max.connections.per.ip` | 2147483647 | Max connections per IP |
| `connections.max.idle.ms` | 600000 | Close idle connections after |

### Message Configuration

| Config | Default | Description |
|--------|---------|-------------|
| `message.max.bytes` | 1048588 | Max message size (broker level) |
| `replica.fetch.max.bytes` | 1048576 | Max bytes per replica fetch |

---

## Troubleshooting

### Config Change Not Taking Effect

Some configs require specific conditions:

```bash
# Check if config is read-only
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | grep <config-name>
```

Look at the source - `READ_ONLY` configs cannot be changed dynamically.

### Finding Valid Config Names

List all broker configs and their current values:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-name 1 \
  --describe --all 2>/dev/null | less
```

### Invalid Config Value

If you set an invalid value, Kafka will reject it:

```bash
# This will fail - negative value not allowed
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server kafka-1:19092 \
  --entity-type brokers \
  --entity-default \
  --alter \
  --add-config num.io.threads=-1
```

### Viewing Config Source

The `--all` flag shows where each config value comes from:

- `DYNAMIC_BROKER_CONFIG` - Per-broker dynamic override
- `DYNAMIC_DEFAULT_BROKER_CONFIG` - Cluster-wide dynamic default
- `STATIC_BROKER_CONFIG` - From server.properties
- `DEFAULT_CONFIG` - Kafka default value

---

## Best Practices

1. **Document before changing**: Always record current config before modifications
2. **Canary deployments**: Test changes on one broker before applying cluster-wide
3. **Monitor after changes**: Watch metrics for unexpected behavior
4. **Use cluster-wide when possible**: Easier to manage than per-broker overrides
5. **Keep per-broker configs minimal**: Only use when brokers truly need different values
6. **Prefer milliseconds over hours**: More precise control (use `log.retention.ms` over `log.retention.hours`)
7. **Have rollback ready**: Know the command to revert before applying changes
8. **Audit changes**: Keep a log of who changed what and when
9. **Test in non-production first**: Validate behavior in a safe environment
10. **Understand config interactions**: Some configs affect others (e.g., thread counts and CPU usage)

---

## Cleanup

Stop all services:

```bash
docker compose down -v
```

Remove backup files:

```bash
rm -f config-backup.txt
```

---

## Additional Resources

- [Kafka Broker Configurations](https://kafka.apache.org/documentation/#brokerconfigs)
- [Kafka Topic Configurations](https://kafka.apache.org/documentation/#topicconfigs)
- [KIP-226: Dynamic Broker Configuration](https://cwiki.apache.org/confluence/display/KAFKA/KIP-226+-+Dynamic+Broker+Configuration)

## Next Steps

Try these challenges:

1. Create a script that backs up all dynamic configs to a file
2. Implement a config diff tool that compares two brokers
3. Set up different retention policies for different topics
4. Tune thread counts based on your machine's CPU cores
5. Create a runbook for common configuration changes in your environment