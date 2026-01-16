# Exercise 7.05: Broker Maintenance & Graceful Shutdown

## Learning Objectives

- Understand how Kafka handles broker shutdown automatically (controlled shutdown)
- Learn the difference between leadership election and partition reassignment
- Practice verifying cluster health before and after maintenance
- Understand when manual intervention is needed vs. relying on Kafka's built-in mechanisms
- Master pre-maintenance and post-maintenance verification steps

## Background

### Kafka's Built-in Graceful Shutdown

Kafka has a feature called **controlled shutdown** (enabled by default via `controlled.shutdown.enable=true`). When a broker shuts down gracefully:

1. The broker notifies the controller it's shutting down
2. The controller moves leadership of affected partitions to other in-sync replicas
3. The broker waits for leadership transfers to complete
4. Only then does the broker fully stop

This means **for most maintenance scenarios, you don't need to do anything special** - just stop the broker gracefully and Kafka handles leadership transfer automatically.

### So Why This Exercise?

While controlled shutdown works well, there are reasons to understand manual approaches:

| Scenario | Approach |
|----------|----------|
| Normal maintenance | Controlled shutdown (automatic) |
| Want explicit verification before stopping | Manual pre-check + controlled shutdown |
| Controlled shutdown is disabled/broken | Manual leadership migration |
| Need to control timing precisely | Manual approach |
| Rolling upgrade with verification | Manual verification steps |

### Key Concepts

**Leadership Election** (metadata-only, fast):
- Changes which replica serves as leader
- No data movement
- Uses `kafka-leader-election.sh`

**Partition Reassignment** (data movement, slow):
- Changes which brokers hold replicas
- Involves copying data between brokers
- Uses `kafka-reassign-partitions.sh`
- **NOT needed for routine maintenance**

**Preferred Replica**:
- The first broker in a partition's replica list
- Kafka prefers this replica as leader
- Used for balanced leadership distribution

### Important Distinction

```
Replica list: [2, 3, 1]
               ▲
               └── Preferred replica (broker 2)

Leadership election: Pick which of [2, 3, 1] is leader (no data movement)
Partition reassignment: Change the list itself (may involve data movement)
```

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Kafka Cluster                                     │
│                                                                           │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                  │
│   │  Broker 1   │    │  Broker 2   │    │  Broker 3   │                  │
│   │             │    │   (TARGET)  │    │             │                  │
│   │  orders-0 F │    │  orders-0 L │    │  orders-0 F │                  │
│   │  orders-1 F │    │  orders-1 F │    │  orders-1 L │                  │
│   │  orders-2 L │    │  orders-2 F │    │  orders-2 F │                  │
│   └─────────────┘    └─────────────┘    └─────────────┘                  │
│                                                                           │
│   L = Leader, F = Follower                                               │
│                                                                           │
│   When Broker 2 shuts down gracefully:                                   │
│   - orders-0 leadership moves to Broker 1 or 3 (whichever is in ISR)    │
│   - No manual intervention needed                                        │
│   - Clients automatically reconnect to new leader                        │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Completed exercises 1.01-1.06 (Kafka fundamentals)
- Docker and Docker Compose installed

## Setup

Start the 3-broker cluster:

```bash
docker compose up -d
```

Wait about 15 seconds for the cluster to initialize.

Verify all brokers are running:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server kafka-1:19092 2>&1 | grep -E "^kafka-[0-9]"
```

You should see all three brokers listed.

---

## Part 1: Understanding Controlled Shutdown

### Task 1: Create Test Topics

Create a topic with replication across all brokers:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic orders \
  --bootstrap-server kafka-1:19092 \
  --partitions 6 \
  --replication-factor 3
```

### Task 2: Produce Test Data

Add some data to verify integrity later:

```bash
for i in {1..1000}; do
  echo "order-$i"
done | docker exec -i kafka-1 /opt/kafka/bin/kafka-console-producer.sh \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

### Task 3: Examine Current Leadership

Check which broker leads which partitions:

```bash
echo "=== Current Partition Leadership ==="
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Count leaders per broker:

```bash
echo "=== Leaders per Broker ==="
for broker in 1 2 3; do
  count=$(docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
    --describe \
    --bootstrap-server kafka-1:19092 2>/dev/null | grep "Leader: $broker" | wc -l)
  echo "Broker $broker: $count leader partitions"
done
```

Note how many partitions Broker 2 leads - we'll see this change after shutdown.

### Task 4: Verify Cluster Health

Before any maintenance, always check for under-replicated partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --under-replicated-partitions \
  --bootstrap-server kafka-1:19092
```

**If this returns any partitions, do NOT proceed with maintenance.**

### Task 5: Graceful Shutdown (Controlled Shutdown in Action)

Stop Broker 2 gracefully:

```bash
docker compose stop kafka-2
```

Docker Compose sends SIGTERM, which triggers Kafka's controlled shutdown.

### Task 6: Observe Automatic Leadership Transfer

Check leadership distribution immediately after shutdown:

```bash
echo "=== Leadership After Broker 2 Shutdown ==="
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Notice:
- Broker 2 is no longer leader for any partition
- Leadership automatically moved to other replicas in the ISR
- **You didn't have to do anything!**

Count leaders now:

```bash
echo "=== Leaders per Broker (After Shutdown) ==="
for broker in 1 2 3; do
  count=$(docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
    --describe \
    --bootstrap-server kafka-1:19092 2>/dev/null | grep "Leader: $broker" | wc -l)
  echo "Broker $broker: $count leader partitions"
done
```

Broker 2 should show 0 leaders.

### Task 7: Verify Data Is Still Accessible

Even with Broker 2 down, all data is accessible:

```bash
echo "=== Message Count (Broker 2 is DOWN) ==="
docker exec kafka-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic orders \
  --bootstrap-server kafka-1:19092 \
  --from-beginning \
  --timeout-ms 5000 2>/dev/null | wc -l
```

Should still show 1000 messages.

### Task 8: Restart the Broker

Simulate completing maintenance and restarting:

```bash
docker compose start kafka-2
```

Wait for the broker to rejoin:

```bash
sleep 15
```

### Task 9: Verify Broker Has Rejoined

Check the broker is back:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server kafka-1:19092 2>&1 | grep -E "^kafka-[0-9]"
```

### Task 10: Check ISR Status

Verify Broker 2 is back in all ISRs:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Look at the ISR column - Broker 2 should be listed for all partitions.

Check for under-replicated partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --under-replicated-partitions \
  --bootstrap-server kafka-1:19092
```

Should return nothing once Broker 2 has caught up.

### Task 11: Restore Balanced Leadership

After Broker 2 rejoins, it's a follower for all partitions. To restore balanced leadership, trigger a preferred leader election:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-leader-election.sh \
  --bootstrap-server kafka-1:19092 \
  --election-type preferred \
  --all-topic-partitions
```

Check the distribution:

```bash
echo "=== Leaders per Broker (After Election) ==="
for broker in 1 2 3; do
  count=$(docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
    --describe \
    --bootstrap-server kafka-1:19092 2>/dev/null | grep "Leader: $broker" | wc -l)
  echo "Broker $broker: $count leader partitions"
done
```

Leadership should be balanced again.

---

## Part 2: Manual Leadership Migration (Extra Caution)

Sometimes you want explicit control - perhaps controlled shutdown has issues, or you want to verify the state before stopping the broker.

### Task 12: Identify Partitions Led by Broker 2

```bash
echo "=== Partitions with Broker 2 as Leader ==="
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --bootstrap-server kafka-1:19092 | grep "Leader: 2"
```

### Task 13: Understanding the Challenge

Here's the problem: `kafka-leader-election.sh` supports:
- `--election-type preferred`: Elects the preferred (first) replica as leader
- `--election-type unclean`: Elects any replica, even if not in ISR (dangerous!)

Neither directly says "move leadership away from broker X".

To manually move leadership away from a broker, you have two options:

**Option A: Reorder the replica list (change preferred replica)**

Change `[2, 3, 1]` to `[3, 2, 1]` - same replicas, different order. Since all replicas already exist on these brokers, **no data movement occurs**. This is a metadata-only operation.

**Option B: Rely on controlled shutdown**

Just stop the broker - Kafka will elect a new leader from the remaining ISR members.

### Task 14: Reorder Preferred Replicas (Option A)

First, see current assignments:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Create a reassignment that reorders replicas to put Broker 2 last (but keeps the same set of brokers):

```bash
cat > preferred-reorder.json << 'EOF'
{
  "version": 1,
  "partitions": [
    {"topic": "orders", "partition": 0, "replicas": [1, 3, 2]},
    {"topic": "orders", "partition": 1, "replicas": [3, 1, 2]},
    {"topic": "orders", "partition": 2, "replicas": [1, 3, 2]},
    {"topic": "orders", "partition": 3, "replicas": [3, 1, 2]},
    {"topic": "orders", "partition": 4, "replicas": [1, 3, 2]},
    {"topic": "orders", "partition": 5, "replicas": [3, 1, 2]}
  ]
}
EOF

docker cp preferred-reorder.json kafka-1:/tmp/preferred-reorder.json
```

Execute the reordering:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/preferred-reorder.json \
  --execute
```

**Note**: This completes almost instantly because no data is being moved - just the replica order in metadata.

### Task 15: Trigger Preferred Leader Election

Now elect the new preferred replicas as leaders:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-leader-election.sh \
  --bootstrap-server kafka-1:19092 \
  --election-type preferred \
  --all-topic-partitions
```

### Task 16: Verify Broker 2 Has No Leadership

```bash
echo "=== Partitions with Broker 2 as Leader (should be empty) ==="
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --bootstrap-server kafka-1:19092 | grep "Leader: 2"

echo ""
echo "=== Leaders per Broker ==="
for broker in 1 2 3; do
  count=$(docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
    --describe \
    --bootstrap-server kafka-1:19092 2>/dev/null | grep "Leader: $broker" | wc -l)
  echo "Broker $broker: $count leader partitions"
done
```

Broker 2 should have 0 leader partitions.

### Task 17: Now Safe to Stop Broker 2

```bash
docker compose stop kafka-2
```

Since Broker 2 wasn't leading any partitions, this has **zero impact** on clients.

### Task 18: Complete Maintenance and Restart

```bash
# Simulate maintenance
sleep 5

# Restart
docker compose start kafka-2

# Wait for rejoin
sleep 15
```

### Task 19: Verify and Restore

Check Broker 2 is back in ISR:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --under-replicated-partitions \
  --bootstrap-server kafka-1:19092
```

Restore balanced replica order:

```bash
cat > restore-order.json << 'EOF'
{
  "version": 1,
  "partitions": [
    {"topic": "orders", "partition": 0, "replicas": [1, 2, 3]},
    {"topic": "orders", "partition": 1, "replicas": [2, 3, 1]},
    {"topic": "orders", "partition": 2, "replicas": [3, 1, 2]},
    {"topic": "orders", "partition": 3, "replicas": [1, 2, 3]},
    {"topic": "orders", "partition": 4, "replicas": [2, 3, 1]},
    {"topic": "orders", "partition": 5, "replicas": [3, 1, 2]}
  ]
}
EOF

docker cp restore-order.json kafka-1:/tmp/restore-order.json

docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/restore-order.json \
  --execute
```

Trigger preferred leader election:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-leader-election.sh \
  --bootstrap-server kafka-1:19092 \
  --election-type preferred \
  --all-topic-partitions
```

Verify balanced leadership:

```bash
echo "=== Final Leaders per Broker ==="
for broker in 1 2 3; do
  count=$(docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
    --describe \
    --bootstrap-server kafka-1:19092 2>/dev/null | grep "Leader: $broker" | wc -l)
  echo "Broker $broker: $count leader partitions"
done
```

---

## Comparing Approaches

| Aspect | Controlled Shutdown | Manual Leadership Migration |
|--------|--------------------|-----------------------------|
| **Complexity** | Simple - just stop the broker | More steps required |
| **Speed** | Fast | Slightly slower (election step) |
| **Client Impact** | Brief blip during leader election | None if done before stopping |
| **Verification** | After the fact | Before stopping broker |
| **Use Case** | Normal maintenance | Extra caution, broken controlled shutdown |

### When to Use Each

**Use Controlled Shutdown (Part 1) when:**
- Normal maintenance windows
- You trust Kafka's built-in mechanisms
- Speed is important
- Low-risk changes (restarts, config changes)

**Use Manual Migration (Part 2) when:**
- High-stakes maintenance (firmware updates, hardware changes)
- You need to verify leadership moved before stopping
- Controlled shutdown is disabled or unreliable
- You want explicit control over timing

---

## Production Maintenance Checklist

### Pre-Maintenance

- [ ] Check cluster health (no under-replicated partitions)
- [ ] Identify partitions led by target broker
- [ ] Notify stakeholders of maintenance window
- [ ] Ensure monitoring is active
- [ ] (Optional) Manually migrate leadership for extra safety

### During Maintenance

- [ ] Stop the broker gracefully (SIGTERM, not SIGKILL)
- [ ] Verify leadership transferred (for controlled shutdown)
- [ ] Perform maintenance tasks
- [ ] Start the broker

### Post-Maintenance

- [ ] Verify broker rejoined cluster
- [ ] Wait for ISR sync (no under-replicated partitions)
- [ ] Trigger preferred leader election for balance
- [ ] Verify data integrity (sample consumers)
- [ ] Document completion

---

## Rolling Restart Procedure

For cluster-wide maintenance (e.g., version upgrades):

```
For each broker in cluster:
  1. Verify cluster health (no under-replicated partitions)
  2. (Optional) Move leadership away from broker
  3. Stop broker gracefully
  4. Perform maintenance (upgrade, config change, etc.)
  5. Start broker
  6. Wait for broker to rejoin and sync
  7. Verify no under-replicated partitions
  8. Proceed to next broker
  
After all brokers:
  9. Trigger preferred leader election
  10. Verify balanced leadership distribution
```

**Critical**: Always wait for a broker to fully sync before proceeding to the next one!

---

## Key Concepts Summary

### Leadership Election vs. Partition Reassignment

| Operation | Tool | Data Movement | Speed | Use Case |
|-----------|------|---------------|-------|----------|
| Leader Election | `kafka-leader-election.sh` | No | Fast | Change which replica is leader |
| Replica Reordering | `kafka-reassign-partitions.sh` (same brokers) | No | Fast | Change preferred replica |
| Replica Movement | `kafka-reassign-partitions.sh` (different brokers) | Yes | Slow | Add/remove/move replicas |

### Controlled Shutdown Configuration

```properties
# Enable graceful shutdown (default: true)
controlled.shutdown.enable=true

# Max retries for leadership transfer (default: 3)
controlled.shutdown.max.retries=3

# Retry backoff (default: 5000ms)
controlled.shutdown.retry.backoff.ms=5000
```

### Auto Leader Rebalancing

```properties
# Automatically restore preferred leaders (default: true)
auto.leader.rebalance.enable=true

# Check interval (default: 300 seconds)
leader.imbalance.check.interval.seconds=300

# Imbalance threshold to trigger rebalance (default: 10%)
leader.imbalance.per.broker.percentage=10
```

With auto-rebalancing enabled, leadership will automatically return to preferred replicas after a broker rejoins - you don't need to manually trigger election.

---

## Troubleshooting

### Controlled Shutdown Takes Too Long

Check broker logs:

```bash
docker logs kafka-2 | grep -i "shutdown\|leader"
```

Common causes:
- Many partitions to transfer
- Slow ISR sync
- Controller issues

### Broker Won't Rejoin After Restart

```bash
docker logs kafka-2 | tail -50
```

Common issues:
- Cluster ID mismatch
- Port conflicts
- Network connectivity
- Corrupted logs

### Under-Replicated Partitions After Restart

Give the broker time to catch up:

```bash
# Wait and check again
sleep 30
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --under-replicated-partitions \
  --bootstrap-server kafka-1:19092
```

If persists, check disk I/O and network bandwidth.

### Leadership Not Balanced After Restart

If auto-rebalancing is disabled or you want immediate balance:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-leader-election.sh \
  --bootstrap-server kafka-1:19092 \
  --election-type preferred \
  --all-topic-partitions
```

---

## Cleanup

Stop all services:

```bash
docker compose down -v
```

Remove temporary files:

```bash
rm -f preferred-reorder.json restore-order.json
```

---

## Best Practices

1. **Trust controlled shutdown** for routine maintenance - it works well
2. **Always verify cluster health** before starting maintenance
3. **Wait for ISR sync** after broker restart before proceeding
4. **One broker at a time** for rolling operations
5. **Monitor during maintenance** - watch for under-replicated partitions
6. **Test procedures** in non-production first
7. **Document everything** - keep a maintenance log
8. **Prefer graceful stops** - SIGTERM, not SIGKILL
9. **Enable auto-rebalancing** unless you have a specific reason not to
10. **Communicate** with stakeholders about maintenance windows

---

## Additional Resources

- [Kafka Operations - Graceful Shutdown](https://kafka.apache.org/documentation/#basic_ops_restarting)
- [Kafka Operations - Balancing Leadership](https://kafka.apache.org/documentation/#basic_ops_leader_balancing)
- [KIP-762: Graceful Broker Shutdown Improvements](https://cwiki.apache.org/confluence/display/KAFKA/KIP-762%3A+Graceful+broker+shutdown)

## Next Steps

Try these challenges:

1. Disable controlled shutdown and observe what happens on broker stop
2. Simulate a hard failure (SIGKILL) and compare to graceful shutdown
3. Practice rolling restart across all 3 brokers
4. Monitor the cluster during maintenance using Kafka UI (http://localhost:8080)
5. Write a script that automates the pre-maintenance checklist