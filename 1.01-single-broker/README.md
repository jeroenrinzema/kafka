# Exercise 1.01: Single Broker Setup

## Learning Objectives

- Understand Kafka's basic architecture
- Start a single Kafka broker using KRaft mode (without ZooKeeper)
- Verify the broker is running correctly

## Background

Apache Kafka is a distributed event streaming platform. In this exercise, you'll set up the simplest possible Kafka cluster: a single broker running in KRaft mode (Kafka's new consensus protocol that replaces ZooKeeper).

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Verify the broker is running:
```bash
docker compose ps
```

## Tasks

### Task 1: Connect to the Kafka Broker

Execute a shell inside the broker container:
```bash
docker exec -it broker bash
```

### Task 2: List Available Kafka Tools

Inside the container, navigate to the Kafka binaries:
```bash
cd /opt/kafka/bin
ls kafka-*
```

You should see various Kafka command-line tools. These will be your primary tools for the next few exercises.

### Task 3: Explore the Broker Metadata

Use the kafka-metadata tool to inspect the broker:
```bash
./kafka-broker-api-versions.sh --bootstrap-server localhost:9092
```

This shows the API versions supported by your broker.

### Task 4: Check Cluster ID

The cluster ID is a unique identifier for your Kafka cluster:
```bash
./kafka-cluster.sh cluster-id --bootstrap-server localhost:9092
```

## Verification

You've successfully completed this exercise when:
- ✅ The broker container is running
- ✅ You can execute commands inside the broker container
- ✅ You can retrieve the cluster ID

## Cleanup

To stop the broker:
```bash
docker compose -f 01-single-broker.yaml down
```

To remove all data:
```bash
docker compose -f 01-single-broker.yaml down -v
```

## Next Steps

Continue to [Exercise 1.02: Topic Creation](../1.02-topic-creation/)
