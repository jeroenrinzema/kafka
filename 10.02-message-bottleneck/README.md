# Exercise 10.02: Message Bottleneck Challenge

## Learning Objectives

- Optimize topic configuration for high-throughput scenarios
- Understand the relationship between partitions, brokers, and consumers

## Background

You've been asked to review a metrics collection system that's experiencing performance issues. The system collects metrics from multiple regions (us-east, us-west, eu-west, ap-south) and needs to handle high throughput.

The infrastructure team has set up a 3-broker Kafka cluster for reliability, but users are reporting that metric ingestion is slower than expected.

## Scenario

The current setup:
- **3-broker Kafka cluster** (for high availability)
- **Metrics topic** collecting data from 4 regions
- **Producer** sending 1000 metrics as fast as possible
- **Target**: Process metrics quickly with good distribution

**The Problem**: Despite having 3 brokers, the system isn't performing as well as expected. Your task is to figure out why and fix it!

## Setup

1. Start the 3-broker Kafka cluster:
```bash
docker compose up -d
```

Wait for all brokers to start (about 10-15 seconds).

2. Access Kafka UI at http://localhost:8080 to visualize the cluster

3. Create the metrics topic
```bash
docker exec -it broker-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --topic metrics \
  --replication-factor 3
```

4. Run the producer and observe the performance:
```bash
cd producer
go run main.go
```

Observe the throughput reported. Is it as high as you'd expect with 3 brokers?

## Your Challenge

Something is wrong with the performance. Your job is to investigate and fix it!

**Questions to answer:**
- Why is the throughput low despite having 3 brokers?
- Where is the bottleneck?
- How can you improve the performance?

## Debugging Tools

**Kafka UI (Visual):**
- **Dashboard**: http://localhost:8080
- **Topics view**: See partition count, replication, size
- **Messages**: Consume and see which partition each message is in
- **Brokers**: View all 3 brokers and their status

**CLI Tools:**
```bash
# List all brokers
docker exec -it broker-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092 | grep id

# Check topic details
docker exec -it broker-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic metrics
```

**Monitor partition distribution:**
```bash
# See which partitions each consumer would handle
docker exec -it broker-1 /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --all-groups
```

### CLI Tools

**Inspect topic configuration:**
```bash
docker exec -it broker-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic metrics
```

**Consume messages with partition info:**
```bash
docker exec -it broker-1 /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic metrics \
  --from-beginning \
  --max-messages 20 \
  --property print.partition=true \
  --property print.key=true
```

**Delete a topic:**
```bash
docker exec -it broker-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --delete \
  --topic metrics
```

**Create a topic:**
```bash
docker exec -it broker-1 /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --topic metrics \
  --partitions <number> \
  --replication-factor 3
```

**List all brokers:**
```bash
docker exec -it broker-1 /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9092 | grep id
```
