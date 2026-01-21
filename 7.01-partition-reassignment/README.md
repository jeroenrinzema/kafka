# Exercise 7.01: Partition Reassignment & Broker Decommissioning

## Learning Objectives

- Understand how partitions are distributed across brokers
- Generate and execute partition reassignment plans
- Throttle reassignment to minimize cluster impact
- Safely decommission a broker
- Add a new broker and rebalance partitions
- Verify data integrity after reassignment

## Background

Partition reassignment is one of the most common administrative tasks in Kafka. You'll need it when:

- **Scaling out**: Adding new brokers and distributing load
- **Scaling in**: Decommissioning brokers for cost savings
- **Rebalancing**: Fixing uneven partition distribution
- **Hardware replacement**: Moving data off failing hardware
- **Rack awareness**: Ensuring replicas are spread across failure domains

### How Partition Reassignment Works

1. **Plan Generation**: Create a JSON file describing the new partition layout
2. **Execution**: Kafka creates new replicas on target brokers
3. **Replication**: Data is copied from existing replicas to new ones
4. **Completion**: Old replicas are removed, leaders are elected on new replicas

### Key Concepts

- **Preferred Replica**: The first replica in the list, preferred as leader
- **Leader Election**: After reassignment, preferred replica becomes leader
- **Throttling**: Limits bandwidth used for reassignment to protect production traffic
- **In-Progress Reassignment**: Can be monitored and cancelled if needed

## Architecture

This exercise uses a 4-broker cluster to demonstrate scaling scenarios.

**Important KRaft Consideration**: In KRaft mode, the controller quorum is fixed at cluster creation. You cannot dynamically add or remove controllers. To scale brokers dynamically, we use:
- **3 combined broker+controller nodes** (fixed quorum)
- **1 broker-only node** (can be added/removed dynamically)

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kafka Cluster (KRaft Mode)                    │
│                                                                  │
│  Controller Quorum (Fixed - 3 nodes)                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Broker 1   │  │  Broker 2   │  │  Broker 3   │              │
│  │ (ID: 1)     │  │ (ID: 2)     │  │ (ID: 3)     │              │
│  │ broker +    │  │ broker +    │  │ broker +    │              │
│  │ controller  │  │ controller  │  │ controller  │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                  │
│  Broker Only (Dynamic - can add/remove)                         │
│  ┌─────────────┐                                                │
│  │  Broker 4   │  ← Added dynamically, no controller role       │
│  │ (ID: 4)     │    Observes quorum but doesn't vote            │
│  │ broker only │                                                │
│  └─────────────┘                                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Completed exercises 1.01-1.06 (Kafka fundamentals)
- Docker and Docker Compose installed

## Tasks

### Task 1: Start the Cluster

Start a 3-broker cluster initially (we'll add broker 4 later):

```bash
docker compose up -d kafka-1 kafka-2 kafka-3 kafka-ui
```

Wait about 15 seconds for the cluster to initialize.

Verify all brokers are running:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server kafka-1:19092 describe --status
```

You should see 3 voters in the `CurrentVoters` list.

### Task 2: Create a Test Topic

Create a topic with partitions spread across the 3 brokers:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic orders \
  --bootstrap-server kafka-1:19092 \
  --partitions 6 \
  --replication-factor 2
```

### Task 3: Examine Current Partition Distribution

View the current partition assignment:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Note which brokers hold which partitions. You'll see output like:

```
Topic: orders   Partition: 0    Leader: 1    Replicas: 1,2    Isr: 1,2
Topic: orders   Partition: 1    Leader: 2    Replicas: 2,3    Isr: 2,3
...
```

### Task 4: Produce Test Data

Generate some test data so we can verify integrity after reassignment:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-producer-perf-test.sh \
  --topic orders \
  --num-records 10000 \
  --record-size 1000 \
  --throughput 1000 \
  --producer-props bootstrap.servers=kafka-1:19092
```

Record the number of messages produced (should be 10,000).

### Task 5: Add a New Broker

Scale up by adding broker 4 (a broker-only node):

```bash
docker compose --profile scale up -d kafka-4
```

Wait 10 seconds, then verify broker 4 has joined:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server kafka-1:19092 describe --replication
```

You should see broker 4 listed as an **Observer** (broker-only nodes observe the quorum but don't vote).

Or check via Kafka UI at http://localhost:8080

### Task 6: Check Current Assignment (Pre-Reassignment)

The new broker has no partitions yet. Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Broker 4 should not appear in any Replicas list.

### Task 7: Generate Reassignment Plan

First, create a JSON file specifying which topics to reassign:

```bash
cat > topics-to-move.json << 'EOF'
{
  "topics": [
    {"topic": "orders"}
  ],
  "version": 1
}
EOF
```

Copy it to the container:

```bash
docker cp topics-to-move.json kafka-1:/tmp/topics-to-move.json
```

Generate a reassignment plan that includes broker 4:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --topics-to-move-json-file /tmp/topics-to-move.json \
  --broker-list "1,2,3,4" \
  --generate
```

This outputs two JSON blocks:
1. **Current assignment** - Save this for rollback!
2. **Proposed assignment** - The new layout including broker 4

### Task 8: Save the Reassignment Plan

Copy the **Proposed partition reassignment configuration** from the previous command output and save it:

```bash
cat > reassignment.json << 'EOF'
<PASTE THE PROPOSED ASSIGNMENT JSON HERE>
EOF
```

For example:
```bash
cat > reassignment.json << 'EOF'
{"version":1,"partitions":[{"topic":"orders","partition":0,"replicas":[1,4],"log_dirs":["any","any"]},{"topic":"orders","partition":1,"replicas":[2,1],"log_dirs":["any","any"]},{"topic":"orders","partition":2,"replicas":[3,2],"log_dirs":["any","any"]},{"topic":"orders","partition":3,"replicas":[4,3],"log_dirs":["any","any"]},{"topic":"orders","partition":4,"replicas":[1,2],"log_dirs":["any","any"]},{"topic":"orders","partition":5,"replicas":[2,3],"log_dirs":["any","any"]}]}
EOF
```

Copy to container:

```bash
docker cp reassignment.json kafka-1:/tmp/reassignment.json
```

### Task 9: Execute Reassignment with Throttling

Execute the reassignment with bandwidth throttling to protect production traffic:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/reassignment.json \
  --throttle 5000000 \
  --execute
```

The `--throttle 5000000` limits reassignment to 5 MB/s per broker.

### Task 10: Monitor Reassignment Progress

Check the status of the reassignment:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/reassignment.json \
  --verify
```

You'll see one of:
- `Reassignment of partition orders-X is still in progress`
- `Reassignment of partition orders-X is completed`

Run this command repeatedly until all partitions show complete. The `--verify` command also clears throttle configs when reassignment is complete.

### Task 11: Verify New Distribution

Check that broker 4 now has partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Broker 4 should now appear in the Replicas lists!

### Task 12: Verify Data Integrity

Consume all messages to verify no data was lost:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-consumer-perf-test.sh \
  --bootstrap-server kafka-1:19092 \
  --topic orders \
  --messages 10000
```

The message count should match what was produced earlier (10,000).

### Task 13: Trigger Preferred Leader Election

After reassignment, the first replica in the list should be the leader (preferred replica). If not, trigger an election:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-leader-election.sh \
  --bootstrap-server kafka-1:19092 \
  --election-type preferred \
  --all-topic-partitions
```

Verify leaders match preferred replicas:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

### Task 14: Decommission a Broker

Now let's remove broker 3 from the cluster. First, move all its partitions away.

Generate a plan excluding broker 3:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --topics-to-move-json-file /tmp/topics-to-move.json \
  --broker-list "1,2,4" \
  --generate
```

Save the proposed assignment:

```bash
cat > decommission.json << 'EOF'
<PASTE THE PROPOSED ASSIGNMENT JSON HERE - excluding broker 3>
EOF

docker cp decommission.json kafka-1:/tmp/decommission.json
```

Execute the decommission plan:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/decommission.json \
  --throttle 10000000 \
  --execute
```

### Task 15: Wait and Verify Decommission

Monitor progress:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/decommission.json \
  --verify
```

Once complete, verify broker 3 has no partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server kafka-1:19092
```

Broker 3 should not appear in any Replicas list.

### Task 16: Understanding Controller Quorum Limitations

**Important**: In KRaft mode, broker 3 is part of the controller quorum. Even though it has no topic partitions, you **cannot remove it from the cluster** without reconfiguring the entire quorum. This is why:

- Controllers (voters) are fixed at cluster creation
- Only broker-only nodes can be truly decommissioned
- In production, use separate controller-only nodes for flexibility

You can stop broker 3's broker role, but the controller will continue running:

```bash
docker compose stop kafka-3
```

> **Note**: This will cause the cluster to lose quorum if you also stop another controller. The cluster needs a majority of controllers (2 out of 3) to function.

### Task 17: Cancel an In-Progress Reassignment (Optional)

If you ever need to cancel a reassignment, use:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/reassignment.json \
  --cancel
```

This stops any in-progress reassignments and rolls back to the original state.

### Task 18: View in Kafka UI

Open http://localhost:8080 and:

1. Navigate to Topics → orders
2. Click on "Partitions" tab
3. Observe the Leader and Replicas for each partition
4. Notice how partitions are distributed across brokers

## Key Concepts

### KRaft Mode and Dynamic Scaling

| Node Type | Can Add Dynamically? | Can Remove Dynamically? |
|-----------|---------------------|------------------------|
| broker + controller | No | No |
| broker only | Yes | Yes |
| controller only | No | No |

For production clusters that need to scale:
- Use **separate controller-only nodes** (typically 3 or 5)
- Add/remove **broker-only nodes** as needed

### Reassignment JSON Format

```json
{
  "version": 1,
  "partitions": [
    {
      "topic": "topic-name",
      "partition": 0,
      "replicas": [1, 2, 3],
      "log_dirs": ["any", "any", "any"]
    }
  ]
}
```

- **replicas**: Ordered list of broker IDs. First is preferred leader.
- **log_dirs**: Optional. Specify exact directories or use "any".

### Throttling Best Practices

| Cluster Size | Recommended Throttle |
|--------------|---------------------|
| Small (< 10 brokers) | 10-50 MB/s |
| Medium (10-50 brokers) | 50-100 MB/s |
| Large (> 50 brokers) | 100-200 MB/s |

Factors to consider:
- Available network bandwidth
- Current cluster load
- How quickly you need the reassignment done
- Time of day (reassign during low-traffic periods)

### Common Pitfalls

1. **Forgetting to remove throttle**: The `--verify` command removes it automatically when complete
2. **Too aggressive throttling**: Reassignment takes forever
3. **No throttling at all**: Can overwhelm network and impact production
4. **Not saving original assignment**: Can't rollback if needed
5. **Stopping broker before reassignment completes**: Data loss risk!
6. **Trying to remove controller nodes**: Not possible without cluster reconfiguration

## Troubleshooting

### Reassignment Stuck

Check for under-replicated partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --under-replicated-partitions \
  --bootstrap-server kafka-1:19092
```

Check broker logs:

```bash
docker logs kafka-4 | grep -i "reassign\|replica"
```

### Throttle Too Low

Increase the throttle by re-executing with a higher value:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server kafka-1:19092 \
  --reassignment-json-file /tmp/reassignment.json \
  --throttle 50000000 \
  --execute
```

### Out of Disk Space

Check disk usage before reassignment:

```bash
docker exec kafka-1 df -h /tmp/kraft-combined-logs
```

During reassignment, new replicas use additional space until old ones are removed.

### Connection Errors

If you see connection errors, make sure you're using the internal network address (`kafka-1:19092`) rather than `localhost:9092` when running commands inside containers.

## Cleanup

Stop all services:

```bash
docker compose --profile scale down -v
```

Remove temporary files:

```bash
rm -f topics-to-move.json reassignment.json decommission.json
```

## Best Practices

1. **Always save the original assignment** before making changes
2. **Use throttling** for any significant data movement
3. **Schedule reassignments during low-traffic periods**
4. **Monitor cluster health** during reassignment
5. **Test the procedure** in a non-production environment first
6. **Verify data integrity** after reassignment completes
7. **Run preferred leader election** after reassignment
8. **Use broker-only nodes** for dynamic scaling in KRaft mode
9. **Plan your controller quorum size** at cluster creation (cannot change later)

## Additional Resources

- [Kafka Operations Documentation](https://kafka.apache.org/documentation/#operations)
- [KIP-455: Replica Reassignment Improvements](https://cwiki.apache.org/confluence/display/KAFKA/KIP-455%3A+Create+an+Administrative+API+for+Replica+Reassignment)
- [KRaft Mode Documentation](https://kafka.apache.org/documentation/#kraft)
- [Confluent Partition Reassignment Guide](https://docs.confluent.io/platform/current/kafka/post-deployment.html#partition-reassignment)

## Next Steps

Try these challenges:

1. Create a 6-broker cluster with 3 controllers and 3 broker-only nodes
2. Implement a script that automatically generates balanced reassignment plans
3. Set up monitoring to alert when reassignment is in progress
4. Practice reassignment with rack-aware placement
5. Combine with rolling restart for version upgrades