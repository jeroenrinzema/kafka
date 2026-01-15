# Apache Kafka Training

A hands-on training for learning Apache Kafka through progressive exercises. This training follows a git-katas style approach where each exercise builds upon previous concepts.

## Prerequisites

- Docker and Docker Compose installed
- Basic understanding of command line
- Text editor of your choice

## Getting Started

Each exercise is contained in its own folder and includes:
- A README with learning objectives and tasks
- Docker Compose configuration to run Kafka
- Sample data or scripts where applicable

Start with exercise 1.01 and progress sequentially through each part.

## Training Structure

### Part 1: Kafka Fundamentals

Core concepts and basic operations with Apache Kafka.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [1.01-single-broker](./1.01-single-broker/) | Single Broker Setup | Setting up a single Kafka broker in KRaft mode |
| [1.02-topic-creation](./1.02-topic-creation/) | Topic Creation | Creating and describing topics |
| [1.03-producer-basics](./1.03-producer-basics/) | Producer Basics | Producing messages to Kafka |
| [1.04-consumer-basics](./1.04-consumer-basics/) | Consumer Basics | Consuming messages from Kafka |
| [1.05-topic-partitions](./1.05-topic-partitions/) | Topic Partitions | Understanding and working with partitions |
| [1.06-offset-management](./1.06-offset-management/) | Offset Management | Consuming from specific offsets and timestamps |

### Part 2: CLI Tools & Consumer Patterns

Advanced consumer patterns and introduction to the Kaf CLI tool.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [2.01-kaf-introduction](./2.01-kaf-introduction/) | Kaf CLI Introduction | Introduction to the Kaf CLI tool |
| [2.02-batching-and-commits](./2.02-batching-and-commits/) | Batching & Commits | Understanding auto-commit and manual offset management |
| [2.03-retry-mechanism](./2.03-retry-mechanism/) | Retry Mechanism | Error handling and retry strategies with Dead Letter Queues |

### Part 3: Schema Management & Data Integration

Managing message schemas and integrating external data sources with Kafka Connect.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [3.01-schema-registry](./3.01-schema-registry/) | Schema Registry | Managing schemas with Confluent Schema Registry |
| [3.02-schema-registry-client](./3.02-schema-registry-client/) | Schema Registry Client | Producing and consuming with Schema Registry in Go |
| [3.03-connect-fundamentals](./3.03-connect-fundamentals/) | Kafka Connect Fundamentals | Kafka Connect architecture, REST API, source and sink connectors |
| [3.04-debezium-cdc](./3.04-debezium-cdc/) | Change Data Capture | Capturing database changes with Debezium PostgreSQL connector |

### Part 4: Event Design & Stream Processing

Event modeling patterns and stream processing with ksqlDB.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [4.01-event-design](./4.01-event-design/) | Event Design | Event modeling best practices and schema design patterns |
| [4.02-compacted-topics](./4.02-compacted-topics/) | Compacted Topics | Using log compaction for state stores and change data capture |
| [4.03-ksqldb-basics](./4.03-ksqldb-basics/) | ksqlDB Basics | Stream processing with ksqlDB |

### Part 5: Security

Securing Kafka with authentication, authorization, and encryption.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [5.01-access-control-lists](./5.01-access-control-lists/) | Access Control Lists | Securing Kafka with SASL authentication and ACL authorization |
| [5.02-ssl-encryption](./5.02-ssl-encryption/) | SSL/TLS Encryption | Encrypting Kafka communication with SSL/TLS and mutual TLS (mTLS) |

### Part 6: Monitoring & Observability

Setting up monitoring and observability for Kafka clusters.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [6.01-grafana-monitoring](./6.01-grafana-monitoring/) | Grafana Monitoring | Monitoring Kafka with Prometheus and Grafana |

### Part 7: Cluster Administration

Essential skills for managing and operating Kafka clusters.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [7.01-partition-reassignment](./7.01-partition-reassignment/) | Partition Reassignment | Moving partitions between brokers, scaling clusters, and decommissioning nodes |
| [7.02-client-quotas](./7.02-client-quotas/) | Client Quotas | Rate limiting producers and consumers with quotas and throttling |
| [7.03-log-segment-inspection](./7.03-log-segment-inspection/) | Log Segment Inspection | Forensic analysis of Kafka log segments and index files |
| [7.04-partition-scaling](./7.04-partition-scaling/) | Partition Scaling | Reducing partition counts using the Create-Migrate-Delete pattern |
| [7.05-broker-maintenance](./7.05-broker-maintenance/) | Broker Maintenance | Zero-downtime broker restarts with graceful leadership migration |
| [7.06-dynamic-broker-config](./7.06-dynamic-broker-config/) | Dynamic Broker Config | Hot-reloading broker configurations without restarts |

### Part 8: Disaster Recovery

Backup strategies and cross-cluster replication.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [8.01-mirrormaker-backup](./8.01-mirrormaker-backup/) | MirrorMaker Backup | Topic backup and cross-cluster replication with MirrorMaker 2 |
| [8.02-s3-backup](./8.02-s3-backup/) | S3 Backup | Long-term topic archival to S3-compatible storage with Kafka Connect |

### Part 9: Resilience Testing

Chaos engineering and failure testing for Kafka clusters.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [9.01-chaos-broker-failure](./9.01-chaos-broker-failure/) | Chaos Engineering | Testing broker failures and cluster resilience |

### Part 10: Troubleshooting (Capstone)

Apply everything you've learned to debug and optimize Kafka systems.

| Exercise | Topic | Description |
|----------|-------|-------------|
| [10.01-debugging-challenge](./10.01-debugging-challenge/) | Debugging Challenge | Troubleshooting common Kafka issues |
| [10.02-message-bottleneck](./10.02-message-bottleneck/) | Message Bottleneck | Identifying and resolving performance bottlenecks |

## Learning Path

Follow the exercises in order. Each part builds on concepts from previous parts:

1. **Part 1-2**: Foundation — Learn core Kafka concepts and CLI tools
2. **Part 3-4**: Application Development — Schema management and event design
3. **Part 5-6**: Production Readiness — Security and monitoring
4. **Part 7**: Operations — Cluster administration skills
5. **Part 8-9**: Reliability — Disaster recovery and resilience testing
6. **Part 10**: Mastery — Apply all skills to real-world debugging scenarios

## Tools Used

- **Part 1**: Official Apache Kafka binaries (kafka-topics, kafka-console-producer, kafka-console-consumer)
- **Part 2**: [Kaf](https://github.com/birdayz/kaf) - A modern CLI for Apache Kafka
- **Part 3**: [Confluent Schema Registry](https://docs.confluent.io/platform/current/schema-registry/) - Schema management and validation; [Kafka Connect](https://kafka.apache.org/documentation/#connect) - Data integration framework; [Debezium](https://debezium.io/) - Change Data Capture
- **Part 4**: [ksqlDB](https://ksqldb.io/) - Streaming SQL engine for Kafka
- **Part 6**: [Prometheus](https://prometheus.io/) & [Grafana](https://grafana.com/) - Monitoring stack

## Additional Resources

- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [Kafka Connect Documentation](https://kafka.apache.org/documentation/#connect)
- [Debezium Documentation](https://debezium.io/documentation/)
- [Kaf CLI Documentation](https://github.com/birdayz/kaf)
- [Confluent Documentation](https://docs.confluent.io/)
- [ksqlDB Documentation](https://docs.ksqldb.io/)