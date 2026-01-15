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

Start with exercise 01 and progress sequentially.

## Training Structure

### Part 1: Kafka Fundamentals (Using Official Kafka Binaries)

1. **[01-single-broker](./01-single-broker/)** - Setting up a single Kafka broker
2. **[02-topic-creation](./02-topic-creation/)** - Creating and describing topics
3. **[03-producer-basics](./03-producer-basics/)** - Producing messages to Kafka
4. **[04-consumer-basics](./04-consumer-basics/)** - Consuming messages from Kafka
5. **[05-topic-partitions](./05-topic-partitions/)** - Understanding and working with partitions
6. **[06-offset-management](./06-offset-management/)** - Consuming from specific offsets and timestamps

### Part 2: Advanced Operations (Using Kaf CLI)

7. **[07-batching-and-commits](./07-batching-and-commits/)** - Understanding auto-commit and manual offset management
8. **[08-retry-mechanism](./08-retry-mechanism/)** - Error handling and retry strategies with Dead Letter Queues
9. **[09-kaf-introduction](./09-kaf-introduction/)** - Introduction to Kaf CLI tool

### Part 3: Schema Management

10. **[10-schema-registry](./10-schema-registry/)** - Managing schemas with Confluent Schema Registry
11. **[11-schema-registry-client](./11-schema-registry-client/)** - Producing and consuming with Schema Registry in Go

### Part 4: Replication and Disaster Recovery

12. **[12-mirrormaker-backup](./12-mirrormaker-backup/)** - Topic backup and cross-cluster replication with MirrorMaker 2
13. **[13-s3-backup](./13-s3-backup/)** - Long-term topic archival to S3-compatible storage with Kafka Connect

### Part 5: Advanced Topics

14. **[14-compacted-topics](./14-compacted-topics/)** - Using log compaction for state stores and change data capture
15. **[15-event-design](./15-event-design/)** - Event modeling best practices and schema design patterns
16. **[16-grafana-monitoring](./16-grafana-monitoring/)** - Monitoring Kafka with Prometheus and Grafana
17. **[17-chaos-broker-failure](./17-chaos-broker-failure/)** - Chaos engineering: testing broker failures and cluster resilience

### Part 6: Security

18. **[18-access-control-lists](./18-access-control-lists/)** - Securing Kafka with SASL authentication and ACL authorization
22. **[22-ssl-encryption](./22-ssl-encryption/)** - Encrypting Kafka communication with SSL/TLS and mutual TLS (mTLS)

### Part 7: Debugging and Performance

19. **[19-ksqldb-basics](./19-ksqldb-basics/)** - Stream processing with ksqlDB
20. **[20-debugging-challenge](./20-debugging-challenge/)** - Troubleshooting common Kafka issues
21. **[21-message-bottleneck](./21-message-bottleneck/)** - Identifying and resolving performance bottlenecks

### Part 8: Cluster Administration

23. **[23-partition-reassignment](./23-partition-reassignment/)** - Moving partitions between brokers, scaling clusters, and decommissioning nodes
24. **[24-client-quotas](./24-client-quotas/)** - Rate limiting producers and consumers with quotas and throttling
25. **[25-log-segment-inspection](./25-log-segment-inspection/)** - Forensic analysis of Kafka log segments and index files
26. **[26-partition-scaling](./26-partition-scaling/)** - Reducing partition counts using the Create-Migrate-Delete pattern
27. **[27-broker-maintenance](./27-broker-maintenance/)** - Zero-downtime broker restarts with graceful leadership migration
28. **[28-dynamic-broker-config](./28-dynamic-broker-config/)** - Hot-reloading broker configurations without restarts

## Learning Path

Follow the exercises in order. Each exercise builds on concepts from previous ones.

## Tools Used

- **Exercises 1-6**: Official Apache Kafka binaries (kafka-topics, kafka-console-producer, kafka-console-consumer)
- **Exercises 7-9**: [Kaf](https://github.com/birdayz/kaf) - A modern CLI for Apache Kafka
- **Exercise 10**: [Confluent Schema Registry](https://docs.confluent.io/platform/current/schema-registry/) - Schema management and validation

## Additional Resources

- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [Kaf CLI Documentation](https://github.com/birdayz/kaf)
