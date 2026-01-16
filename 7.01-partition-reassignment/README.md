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

This exercise uses a 4-broker cluster to demonstrate scaling scenarios:

```
┌─────────────────────────────────────────────────────────────┐
│                    Kafka Cluster                             │
│                                                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        │
│  │ Broker 1│  │ Broker 2│  │ Broker 3│  │ Broker 4│        │
│  │ (ID: 1) │  │ (ID: 2) │  │ (ID: 3) │  │ (ID: 4) │        │
│  │         │  │         │  │         │  │ (new)   │        │
│  │ P0-L    │  │ P0-F    │  │ P1-L    │  │         │        │
│  │ P2-F    │  │ P1-F    │  │ P2-L    │  │         │        │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘        │
│                                                              │
│  L = Leader, F = Follower, P = Partition                    │
└─────────────────────────────────────────────────────────────┘
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
docker exec kafka-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092 | head -5
```

### Task 2: Create a Test Topic

Create a topic with partitions spread across the 3 brokers:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --partitions 6 \
  --replication-factor 2
```

### Task 3: Examine Current Partition Distribution

View the current partition assignment:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server localhost:9092
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
  --producer-props bootstrap.servers=localhost:9092
```

Record the number of messages produced.

### Task 5: Add a New Broker

Scale up by adding broker 4:

```bash
docker compose up -d kafka-4
```

Wait 10 seconds, then verify broker 4 has joined:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-metadata.sh \
  --snapshot /var/kafka-logs/__cluster_metadata-0/00000000000000000000.log \
  --command "broker-ids"
```

Or check via Kafka UI at http://localhost:8080

### Task 6: Check Current Assignment (Pre-Reassignment)

The new broker has no partitions yet. Verify:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server localhost:9092
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
  --bootstrap-server localhost:9092 \
  --topics-to-move-json-file /tmp/topics-to-move.json \
  --broker-list "1,2,3,4" \
  --generate
```

This outputs two JSON blocks:
1. **Current assignment** - Save this for rollback!
2. **Proposed assignment** - The new layout including broker 4

### Task 8: Save the Reassignment Plans

Save the proposed reassignment plan:

```bash
cat > reassignment.json << 'EOF'
{
  "version": 1,
  "partitions": [
    {"topic": "orders", "partition": 0, "replicas": [1, 4]},
    {"topic": "orders", "partition": 1, "replicas": [2, 1]},
    {"topic": "orders", "partition": 2, "replicas": [3, 2]},
    {"topic": "orders", "partition": 3, "replicas": [4, 3]},
    {"topic": "orders", "partition": 4, "replicas": [1, 4]},
    {"topic": "orders", "partition": 5, "replicas": [2, 1]}
  ]
}
EOF
```

> **Note**: Replace the content above with the actual "Proposed partition reassignment configuration" from the previous command.

Copy to container:

```bash
docker cp reassignment.json kafka-1:/tmp/reassignment.json
```

### Task 9: Execute Reassignment with Throttling

Execute the reassignment with bandwidth throttling to protect production traffic:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/reassignment.json \
  --throttle 5000000 \
  --execute
```

The `--throttle 5000000` limits reassignment to 5 MB/s per broker.

### Task 10: Monitor Reassignment Progress

Check the status of the reassignment:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/reassignment.json \
  --verify
```

You'll see one of:
- `Reassignment of partition orders-X is still in progress`
- `Reassignment of partition orders-X is complete`

Run this command repeatedly until all partitions show complete.

### Task 11: Remove Throttling

After reassignment completes, remove the throttle:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/reassignment.json \
  --verify
```

If throttling configs remain, remove them explicitly:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --entity-type brokers \
  --entity-name 1 \
  --alter \
  --delete-config leader.replication.throttled.rate,follower.replication.throttled.rate

docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --entity-type brokers \
  --entity-name 2 \
  --alter \
  --delete-config leader.replication.throttled.rate,follower.replication.throttled.rate

docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --entity-type brokers \
  --entity-name 3 \
  --alter \
  --delete-config leader.replication.throttled.rate,follower.replication.throttled.rate

docker exec kafka-1 /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --entity-type brokers \
  --entity-name 4 \
  --alter \
  --delete-config leader.replication.throttled.rate,follower.replication.throttled.rate
```

### Task 12: Verify New Distribution

Check that broker 4 now has partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server localhost:9092
```

Broker 4 should now appear in the Replicas lists!

### Task 13: Verify Data Integrity

Consume all messages to verify no data was lost:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-consumer-perf-test.sh \
  --broker-list localhost:9092 \
  --topic orders \
  --messages 10000 \
  --print-metrics
```

The message count should match what was produced earlier.

### Task 14: Trigger Preferred Leader Election

After reassignment, the first replica in the list should be the leader (preferred replica). If not, trigger an election:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-leader-election.sh \
  --bootstrap-server localhost:9092 \
  --election-type preferred \
  --topic orders \
  --all-topic-partitions
```

Verify leaders match preferred replicas:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server localhost:9092
```

### Task 15: Decommission a Broker

Now let's remove broker 3 from the cluster. First, move all its partitions away.

Generate a plan excluding broker 3:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --topics-to-move-json-file /tmp/topics-to-move.json \
  --broker-list "1,2,4" \
  --generate
```

Save the proposed assignment and execute it:

```bash
cat > decommission.json << 'EOF'
{
  "version": 1,
  "partitions": [
    {"topic": "orders", "partition": 0, "replicas": [1, 4]},
    {"topic": "orders", "partition": 1, "replicas": [2, 1]},
    {"topic": "orders", "partition": 2, "replicas": [4, 2]},
    {"topic": "orders", "partition": 3, "replicas": [1, 4]},
    {"topic": "orders", "partition": 4, "replicas": [2, 1]},
    {"topic": "orders", "partition": 5, "replicas": [4, 2]}
  ]
}
EOF

docker cp decommission.json kafka-1:/tmp/decommission.json

docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/decommission.json \
  --throttle 10000000 \
  --execute
```

### Task 16: Wait and Verify Decommission

Monitor progress:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/decommission.json \
  --verify
```

Once complete, verify broker 3 has no partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic orders \
  --bootstrap-server localhost:9092
```

Broker 3 should not appear in any Replicas list.

### Task 17: Stop the Decommissioned Broker

Now it's safe to stop broker 3:

```bash
docker compose stop kafka-3
```

Verify the cluster still works:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092
```

### Task 18: Cancel an In-Progress Reassignment (Optional)

If you ever need to cancel a reassignment, use:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/reassignment.json \
  --cancel
```

This stops any in-progress reassignments and rolls back to the original state.

### Task 19: Create a Custom Reassignment Plan

Sometimes you need precise control. Create a manual plan:

```bash
cat > manual-reassignment.json << 'EOF'
{
  "version": 1,
  "partitions": [
    {"topic": "orders", "partition": 0, "replicas": [4, 1], "log_dirs": ["any", "any"]},
    {"topic": "orders", "partition": 1, "replicas": [1, 2], "log_dirs": ["any", "any"]}
  ]
}
EOF
```

The `log_dirs` field allows specifying exact directories on brokers with multiple disks.

### Task 20: View in Kafka UI

Open http://localhost:8080 and:

1. Navigate to Topics → orders
2. Click on "Partitions" tab
3. Observe the Leader and Replicas for each partition
4. Notice how partitions are now spread across brokers 1, 2, and 4

## Key Concepts

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

1. **Forgetting to remove throttle**: Leaves configs on topics and brokers
2. **Too aggressive throttling**: Reassignment takes forever
3. **No throttling at all**: Can overwhelm network and impact production
4. **Not saving original assignment**: Can't rollback if needed
5. **Stopping broker before reassignment completes**: Data loss risk!

## Troubleshooting

### Reassignment Stuck

Check for under-replicated partitions:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --under-replicated-partitions \
  --bootstrap-server localhost:9092
```

Check broker logs:

```bash
docker logs kafka-4 | grep -i "reassign\|replica"
```

### Throttle Too Low

Increase the throttle:

```bash
docker exec kafka-1 /opt/kafka/bin/kafka-reassign-partitions.sh \
  --bootstrap-server localhost:9092 \
  --reassignment-json-file /tmp/reassignment.json \
  --throttle 50000000 \
  --execute
```

### Out of Disk Space

Check disk usage before reassignment:

```bash
docker exec kafka-1 df -h /var/kafka-logs
```

During reassignment, new replicas use additional space until old ones are removed.

## Cleanup

Stop all services:

```bash
docker compose down -v
```

Remove temporary files:

```bash
rm -f topics-to-move.json reassignment.json decommission.json manual-reassignment.json
```

## Best Practices

1. **Always save the original assignment** before making changes
2. **Use throttling** for any significant data movement
3. **Schedule reassignments during low-traffic periods**
4. **Monitor cluster health** during reassignment
5. **Test the procedure** in a non-production environment first
6. **Verify data integrity** after reassignment completes
7. **Run preferred leader election** after reassignment
8. **Clean up throttle configs** when done

## Additional Resources

- [Kafka Operations Documentation](https://kafka.apache.org/documentation/#operations)
- [KIP-455: Replica Reassignment Improvements](https://cwiki.apache.org/confluence/display/KAFKA/KIP-455%3A+Create+an+Administrative+API+for+Replica+Reassignment)
- [Confluent Partition Reassignment Guide](https://docs.confluent.io/platform/current/kafka/post-deployment.html#partition-reassignment)

## Next Steps

Try these challenges:

1. Create a 6-broker cluster and practice large-scale rebalancing
2. Implement a script that automatically generates balanced reassignment plans
3. Set up monitoring to alert when reassignment is in progress
4. Practice reassignment with rack-aware placement
5. Combine with rolling restart for version upgrades