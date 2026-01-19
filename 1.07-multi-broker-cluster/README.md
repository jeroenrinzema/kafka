# Exercise 1.07: Multi-Broker Cluster Setup

## Learning Objectives

- Understand Kafka cluster architecture with multiple brokers
- Configure KRaft mode with multiple controllers and brokers
- Verify cluster health and partition distribution
- Understand quorum voting and leader election
- Test high availability through broker failure scenarios

## Background

In production, Kafka runs as a distributed cluster with multiple brokers for:
- **High Availability**: If one broker fails, others continue serving requests
- **Scalability**: Distribute load across multiple nodes
- **Fault Tolerance**: Replicate data across brokers

With KRaft mode (Kafka Raft), the cluster no longer requires ZooKeeper. Instead, a subset of brokers act as **controllers** that manage cluster metadata using the Raft consensus protocol.

### Cluster Roles

In KRaft mode, each node can have one or more roles:
- **Controller**: Manages cluster metadata, leader election, and configuration
- **Broker**: Stores data and serves client requests
- **Combined**: Acts as both controller and broker (common for smaller clusters)

### Quorum Voters and Consensus

Controllers form a quorum for consensus. The `KAFKA_CONTROLLER_QUORUM_VOTERS` setting defines which nodes participate:
```
1@broker1:29093,2@broker2:29093,3@broker3:29093
```
Format: `nodeId@hostname:controllerPort`

> **Important**: The Raft consensus protocol requires a **majority** of voters to be available for leader election. With 3 voters, at least 2 must be running. This means you cannot start a single broker when the quorum expects 3 voters - all brokers in the quorum must start together.

## Architecture

In this exercise, we'll work with a 3-broker cluster:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kafka Cluster                            │
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Broker 1   │    │  Broker 2   │    │  Broker 3   │         │
│  │ (Controller)│    │ (Controller)│    │ (Controller)│         │
│  │  Node ID: 1 │    │  Node ID: 2 │    │  Node ID: 3 │         │
│  │ Host: 9092  │    │ Host: 9093  │    │ Host: 9094  │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│         │                  │                  │                 │
│         └──────────────────┼──────────────────┘                 │
│                            │                                    │
│                    Raft Consensus                               │
│                  (Controller Quorum)                            │
└─────────────────────────────────────────────────────────────────┘
```

**Port mapping (for host access):**
- Broker 1: localhost:9092 (host) → 9092 (container)
- Broker 2: localhost:9093 (host) → 9092 (container)
- Broker 3: localhost:9094 (host) → 9092 (container)

**Internal network (used by kafka-client):**
- All brokers: port 29092 (PLAINTEXT listener)

## Prerequisites

- Docker and Docker Compose installed
- Completed exercises 1.01 through 1.06

## Tasks

### Task 1: Review the Cluster Configuration

Before starting the cluster, let's understand the configuration. Review `docker-compose.yaml`:

```bash
cat docker-compose.yaml
```

Key configuration elements:
- `KAFKA_NODE_ID`: Unique identifier for this broker (1, 2, 3)
- `KAFKA_PROCESS_ROLES`: What roles this node plays (`broker,controller`)
- `KAFKA_CONTROLLER_QUORUM_VOTERS`: All controllers in the cluster
- `CLUSTER_ID`: Must be the same across all brokers in a cluster
- `kafka-client`: A dedicated container for running CLI commands on the internal network

### Task 2: Start the Cluster

Since KRaft requires a majority of voters for quorum, we start all brokers together:

```bash
docker compose up -d
```

Wait for all brokers to become healthy:

```bash
docker compose ps
```

You should see all 3 brokers with status `healthy` and the `kafka-client` container running. This may take 15-30 seconds.

### Task 3: Verify Cluster Formation

Check the cluster ID (should be the same for all brokers):

```bash
docker exec kafka-client /opt/kafka/bin/kafka-cluster.sh \
  cluster-id --bootstrap-server broker1:29092
```

List all brokers in the cluster using the quorum replication command:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server broker1:29092 describe --replication
```

You should see all 3 nodes listed with their status (one Leader, two Followers).

### Task 4: Check Controller Quorum Status

View the current controller quorum status:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server broker1:29092 describe --status
```

This shows:
- **LeaderId**: Which broker is the active controller
- **LeaderEpoch**: How many leader elections have occurred
- **HighWatermark**: Latest committed offset in the metadata log
- **CurrentVoters**: All nodes participating in the quorum

### Task 5: Create a Replicated Topic

Now that we have multiple brokers, we can create topics with replication:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic replicated-topic \
  --bootstrap-server broker1:29092 \
  --partitions 6 \
  --replication-factor 3
```

Describe the topic to see partition distribution:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic replicated-topic \
  --bootstrap-server broker1:29092
```

Notice how:
- Each partition has a **Leader** (one broker handles reads/writes)
- Each partition has **Replicas** on multiple brokers
- **ISR** (In-Sync Replicas) shows which replicas are caught up

### Task 6: Understand Partition Distribution

Create another topic and observe the distribution:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic distributed-topic \
  --bootstrap-server broker1:29092 \
  --partitions 9 \
  --replication-factor 2
```

```bash
docker exec kafka-client /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic distributed-topic \
  --bootstrap-server broker1:29092
```

With 9 partitions and replication factor 2:
- Each partition exists on 2 brokers
- Leaders should be roughly evenly distributed
- Each broker should lead approximately 3 partitions

### Task 7: Produce Messages to the Cluster

Let's produce some messages to our replicated topic:

```bash
docker exec -it kafka-client /opt/kafka/bin/kafka-console-producer.sh \
  --topic replicated-topic \
  --bootstrap-server broker1:29092 \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Type some messages (key:value format):
```
user1:{"action": "login"}
user2:{"action": "purchase"}
user3:{"action": "logout"}
user1:{"action": "browse"}
user2:{"action": "checkout"}
```
Press Ctrl+D to exit.

### Task 8: Simulate Broker Failure

Now let's test high availability by stopping one broker:

```bash
docker compose stop broker2
```

Check which brokers are still available:

```bash
docker compose ps
```

Check the topic description again:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic replicated-topic \
  --bootstrap-server broker1:29092
```

Notice:
- Leaders have been reassigned away from broker2
- ISR now shows only 2 replicas for affected partitions
- The cluster continues to function!

### Task 9: Consume Messages During Failure

Even with one broker down, we can still consume all messages:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-console-consumer.sh \
  --topic replicated-topic \
  --bootstrap-server broker1:29092 \
  --from-beginning \
  --timeout-ms 5000
```

All messages are still available because they were replicated to multiple brokers.

### Task 10: Recover the Failed Broker

Bring broker2 back:

```bash
docker compose start broker2
```

Wait a few seconds for it to rejoin, then check the topic:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic replicated-topic \
  --bootstrap-server broker1:29092
```

The ISR should now show all 3 replicas again as broker2 catches up.

### Task 11: Observe Controller Failover

Check which broker is currently the active controller:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server broker1:29092 describe --status | grep LeaderId
```

Stop the current controller (if LeaderId is 1, stop broker1; adjust accordingly):

```bash
# Example: if broker1 is the leader
docker compose stop broker1
```

Check the new controller from another broker:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-metadata-quorum.sh \
  --bootstrap-server broker2:29092 describe --status
```

A new controller has been elected automatically! The cluster continues to operate.

Bring the stopped broker back:

```bash
docker compose start broker1
```

### Task 12: Check Log Directories

See how data is distributed across brokers:

```bash
docker exec kafka-client /opt/kafka/bin/kafka-log-dirs.sh \
  --bootstrap-server broker1:29092 \
  --describe \
  --topic-list replicated-topic
```

This shows the size and offset lag for each partition replica on each broker.

## Verification

You've successfully completed this exercise when:
- ✅ You can start a 3-broker cluster and verify all brokers are healthy
- ✅ You understand the controller quorum and can check its status
- ✅ You can create topics with replication across multiple brokers
- ✅ You understand partition distribution (leaders, replicas, ISR)
- ✅ You've tested broker failure and observed automatic failover
- ✅ You've observed controller failover when stopping the active controller
- ✅ You've verified data availability during partial outages

## Key Configuration Summary

| Configuration | Purpose |
|---------------|---------|
| `KAFKA_NODE_ID` | Unique broker identifier |
| `KAFKA_PROCESS_ROLES` | Node roles (broker, controller, or both) |
| `KAFKA_CONTROLLER_QUORUM_VOTERS` | List of all controllers for quorum |
| `CLUSTER_ID` | Unique cluster identifier (must match on all brokers) |
| `KAFKA_LISTENERS` | Network listeners for this broker |
| `KAFKA_ADVERTISED_LISTENERS` | Addresses clients use to connect |
| `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR` | Replication for internal offset topic |
| `KAFKA_MIN_INSYNC_REPLICAS` | Minimum replicas for write acknowledgment |

## Best Practices

1. **Odd number of controllers**: Use 3, 5, or 7 controllers for proper quorum (avoids split-brain)
2. **Separate controller nodes**: In large clusters, dedicate nodes to controller role only
3. **Consistent cluster ID**: All brokers must share the same `CLUSTER_ID`
4. **Replication factor ≥ 3**: For production, always replicate critical data
5. **Min ISR ≥ 2**: Ensure at least 2 replicas acknowledge writes before success
6. **Monitor ISR shrinkage**: Alerts when ISR drops below replication factor

## Troubleshooting

### Cluster won't start / No leader elected

- Ensure a majority of quorum voters are running (e.g., 2 out of 3)
- Verify `CLUSTER_ID` matches across all brokers
- Check `KAFKA_CONTROLLER_QUORUM_VOTERS` includes all controllers with correct hostnames

### Broker won't rejoin after restart

- Check logs: `docker compose logs broker1`
- Ensure the cluster ID hasn't changed
- Verify network connectivity between brokers

### Replication lag / ISR shrinking

- Check disk I/O and network bandwidth
- Monitor with `kafka-topics.sh --describe`
- Consider increasing `replica.fetch.max.bytes`

### Why use kafka-client instead of exec into brokers?

The `kafka-client` container connects via the internal Docker network (PLAINTEXT listener on port 29092). When running commands directly inside a broker container using the PLAINTEXT_HOST listener, metadata returned by Kafka includes `localhost` addresses that only work from the host machine, not from inside containers.

## Cleanup

Stop all containers:

```bash
docker compose down
```

Remove all data:

```bash
docker compose down -v
```

## Next Steps

Continue to [Exercise 2.01: Kaf CLI Introduction](../2.01-kaf-introduction/)