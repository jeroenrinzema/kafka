# Exercise 1.07: Multi-Broker Cluster Setup

## Learning Objectives

- Understand Kafka cluster architecture with multiple brokers
- Configure KRaft mode with multiple controllers and brokers
- Add brokers incrementally to a running cluster
- Verify cluster health and partition distribution
- Understand quorum voting and leader election

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

### Quorum Voters

Controllers form a quorum for consensus. The `KAFKA_CONTROLLER_QUORUM_VOTERS` setting defines which nodes participate:
```
1@broker1:9093,2@broker2:9093,3@broker3:9093
```
Format: `nodeId@hostname:controllerPort`

## Architecture

In this exercise, we'll build a 3-broker cluster step by step:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kafka Cluster                            │
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Broker 1   │    │  Broker 2   │    │  Broker 3   │         │
│  │ (Controller)│    │ (Controller)│    │ (Controller)│         │
│  │  Node ID: 1 │    │  Node ID: 2 │    │  Node ID: 3 │         │
│  │  Port: 9092 │    │  Port: 9093 │    │  Port: 9094 │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│         │                  │                  │                 │
│         └──────────────────┼──────────────────┘                 │
│                            │                                    │
│                    Raft Consensus                               │
│                  (Controller Quorum)                            │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Docker and Docker Compose installed
- Completed exercises 1.01 through 1.06

## Tasks

### Task 1: Start with a Single Broker

First, let's understand how a single broker is configured. Review the `docker-compose.yaml`:

```bash
# Look at the single broker configuration
cat docker-compose.yaml
```

Key configuration elements:
- `KAFKA_NODE_ID`: Unique identifier for this broker (1, 2, 3, etc.)
- `KAFKA_PROCESS_ROLES`: What roles this node plays (broker, controller, or both)
- `KAFKA_CONTROLLER_QUORUM_VOTERS`: All controllers in the cluster
- `CLUSTER_ID`: Must be the same across all brokers in a cluster

Start just the first broker:

```bash
docker compose up -d broker1
```

Verify it's running:

```bash
docker compose ps
```

### Task 2: Explore the Single Broker Cluster

Connect to the broker and check its status:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-metadata.sh \
  --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --command "cat"
```

Check the cluster ID:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-cluster.sh \
  cluster-id --bootstrap-server localhost:9092
```

List brokers (should show only 1):

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092 | grep "id:"
```

### Task 3: Add the Second Broker

Now add broker2 to the cluster:

```bash
docker compose up -d broker2
```

Wait a few seconds for it to join, then verify:

```bash
# Check both containers are running
docker compose ps

# List all brokers in the cluster
docker exec -it broker1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092 | grep "id:"
```

You should now see two brokers listed.

### Task 4: Add the Third Broker

Complete the cluster by adding broker3:

```bash
docker compose up -d broker3
```

Verify all three brokers:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092 | grep "id:"
```

### Task 5: Create a Replicated Topic

Now that we have multiple brokers, we can create topics with replication:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic replicated-topic \
  --bootstrap-server localhost:9092 \
  --partitions 6 \
  --replication-factor 3
```

Describe the topic to see partition distribution:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic replicated-topic \
  --bootstrap-server localhost:9092
```

Notice how:
- Each partition has a **Leader** (one broker)
- Each partition has **Replicas** on multiple brokers
- **ISR** (In-Sync Replicas) shows which replicas are caught up

### Task 6: Understand Partition Distribution

Create another topic and observe the distribution:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic distributed-topic \
  --bootstrap-server localhost:9092 \
  --partitions 9 \
  --replication-factor 2

docker exec -it broker1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic distributed-topic \
  --bootstrap-server localhost:9092
```

With 9 partitions and replication factor 2:
- Each partition exists on 2 brokers
- Leaders should be roughly evenly distributed
- Each broker should lead approximately 3 partitions

### Task 7: Test High Availability

Let's produce some messages and then simulate a broker failure:

```bash
# Produce messages
docker exec -it broker1 /opt/kafka/bin/kafka-console-producer.sh \
  --topic replicated-topic \
  --bootstrap-server localhost:9092 \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Type some messages (key:value format):
```
user1:{"action": "login"}
user2:{"action": "purchase"}
user3:{"action": "logout"}
```
Press Ctrl+D to exit.

Now stop broker2:

```bash
docker compose stop broker2
```

Check the topic description again:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic replicated-topic \
  --bootstrap-server localhost:9092
```

Notice:
- Leaders have been reassigned from broker2
- ISR now shows only 2 replicas for each partition
- The cluster continues to function

### Task 8: Consume Messages During Failure

Even with one broker down, we can still consume:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-console-consumer.sh \
  --topic replicated-topic \
  --bootstrap-server localhost:9092 \
  --from-beginning
```

Press Ctrl+C to exit.

### Task 9: Recover the Failed Broker

Bring broker2 back:

```bash
docker compose start broker2
```

Wait a moment, then check the topic:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-topics.sh \
  --describe \
  --topic replicated-topic \
  --bootstrap-server localhost:9092
```

The ISR should now show all 3 replicas again as broker2 catches up.

### Task 10: Check Controller Status

See which broker is the active controller:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-metadata.sh \
  --snapshot /var/lib/kafka/data/__cluster_metadata-0/00000000000000000000.log \
  --command "cat" | grep -i "controller"
```

Or use the describe command on the quorum:

```bash
docker exec -it broker1 /opt/kafka/bin/kafka-metadata.sh quorum-info \
  --bootstrap-server localhost:9092
```

### Task 11: Test Controller Failover

Stop the current active controller and observe leader election:

```bash
# First, note which broker is the active controller
# Then stop it (assuming broker1 is the controller):
docker compose stop broker1

# Check from another broker
docker exec -it broker2 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server broker2:9092 | grep "id:"
```

A new controller will be elected automatically. Bring broker1 back:

```bash
docker compose start broker1
```

## Verification

You've successfully completed this exercise when:
- ✅ You can start brokers incrementally and see them join the cluster
- ✅ You can create topics with replication across multiple brokers
- ✅ You understand partition distribution across brokers
- ✅ You've tested broker failure and recovery
- ✅ You've observed controller failover

## Key Configuration Summary

| Configuration | Purpose |
|---------------|---------|
| `KAFKA_NODE_ID` | Unique broker identifier |
| `KAFKA_PROCESS_ROLES` | Node roles (broker, controller, or both) |
| `KAFKA_CONTROLLER_QUORUM_VOTERS` | List of all controllers |
| `CLUSTER_ID` | Unique cluster identifier (must match) |
| `KAFKA_LISTENERS` | Network listeners for this broker |
| `KAFKA_ADVERTISED_LISTENERS` | Addresses clients use to connect |

## Best Practices

1. **Odd number of controllers**: Use 3, 5, or 7 controllers for proper quorum
2. **Separate controller nodes**: In large clusters, dedicate nodes to controller role
3. **Consistent cluster ID**: All brokers must share the same `CLUSTER_ID`
4. **Replication factor ≥ 3**: For production, always replicate data
5. **Min ISR = 2**: Ensure at least 2 replicas before acknowledging writes

## Troubleshooting

### Broker won't join cluster

- Verify `CLUSTER_ID` matches across all brokers
- Check `KAFKA_CONTROLLER_QUORUM_VOTERS` includes all controllers
- Ensure network connectivity between brokers

### Controller election stuck

- Check that a majority of controllers are running (2 out of 3)
- Verify controller ports are accessible

### Replication lag

- Check disk I/O and network bandwidth
- Monitor ISR changes with `kafka-topics.sh --describe`

## Cleanup

Stop all brokers:

```bash
docker compose down
```

Remove all data:

```bash
docker compose down -v
```

## Next Steps

Continue to [Exercise 2.01: Kaf CLI Introduction](../2.01-kaf-introduction/)