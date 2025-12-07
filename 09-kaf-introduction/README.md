# Exercise 07: Kaf CLI Introduction

## Learning Objectives

- Install and configure Kaf CLI
- Understand Kaf's improved user experience
- Compare Kaf commands with native Kafka tools
- Use Kaf for common operations
- Leverage Kaf's advanced features

## Background

[Kaf](https://github.com/birdayz/kaf) is a modern CLI for Apache Kafka that provides a more intuitive and user-friendly interface than the official Kafka tools. It offers:
- Simpler command syntax
- Better output formatting
- Interactive features
- Configuration management
- Schema registry support

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Install Kaf on your host machine (macOS):
```bash
brew install kaf
```

For other platforms, see: https://github.com/birdayz/kaf#installation

3. Configure Kaf to connect to your broker:
```bash
kaf config add-cluster local \
  --brokers localhost:9092
```

Set it as the current cluster:
```bash
kaf config use-cluster local
```

Verify configuration:
```bash
kaf config get-cluster
```

## Tasks

### Task 1: List Topics with Kaf

Compare the experience with native Kafka tools:

```bash
# Kafka way
docker exec -it broker kafka-topics --bootstrap-server localhost:9092 --list

# Kaf way
kaf topics
```

**Observe**: Kaf provides a cleaner, more readable output.

### Task 2: Create Topics with Kaf

```bash
kaf topic create users-v2 \
  --partitions 3 \
  --replicas 1
```

Much simpler than the Kafka equivalent!

### Task 3: Describe Topics

```bash
kaf topic describe users-v2
```

Compare with:
```bash
docker exec -it broker kafka-topics --bootstrap-server localhost:9092 --describe --topic users-v2
```

**Observe**: Kaf's output is formatted as a nice table.

### Task 4: Produce Messages with Kaf

Kaf makes producing messages more intuitive:

```bash
kaf produce users-v2
```

Type messages (one per line):
```
{"id": 1, "name": "Alice", "email": "alice@example.com"}
{"id": 2, "name": "Bob", "email": "bob@example.com"}
{"id": 3, "name": "Charlie", "email": "charlie@example.com"}
```

Press `Ctrl+C` when done.

Produce with keys:
```bash
kaf produce users-v2 --key
```

Enter in format `key<tab>value`:
```
1	{"id": 1, "name": "Alice"}
2	{"id": 2, "name": "Bob"}
```

### Task 5: Consume Messages with Kaf

```bash
kaf consume users-v2 --offset oldest
```

**Observe**: 
- Nice formatting
- Colors for better readability
- Automatic display of keys, partitions, offsets

Try consuming from the latest:
```bash
kaf consume users-v2
```

(It will wait for new messages)

### Task 6: Consumer Groups

List consumer groups:
```bash
kaf groups
```

Create a consumer group and consume:
```bash
kaf consume users-v2 --group kaf-demo-group --offset oldest
```

After consuming, check group details:
```bash
kaf group describe kaf-demo-group
```

Much more readable than `kafka-consumer-groups --describe`!

List group members:
```bash
kaf group ls
```

### Task 7: Advanced Consume Options

Consume with filtering (show only specific fields):
```bash
kaf consume users-v2 --offset oldest --raw
```

Consume from specific partition:
```bash
kaf consume users-v2 --partition 0 --offset oldest
```

Consume and follow (like tail -f):
```bash
kaf consume users-v2 --follow
```

### Task 8: Offset Management with Kaf

Reset consumer group offsets:
```bash
kaf group describe kaf-demo-group

# Reset to beginning
kaf group reset kaf-demo-group --topic users-v2 --offset oldest

# Reset to end
kaf group reset kaf-demo-group --topic users-v2 --offset newest
```

### Task 9: Node (Broker) Information

```bash
kaf node ls
```

Get details about nodes:
```bash
kaf node describe
```

### Task 10: Topic Configuration

View topic configuration:
```bash
kaf topic describe users-v2 --config
```

Alter topic configuration:
```bash
kaf topic alter users-v2 --config retention.ms=86400000
```

Verify:
```bash
kaf topic describe users-v2 --config
```

### Task 11: Delete Topics

```bash
kaf topic delete users-v2
```

Confirm when prompted.

### Task 12: Interactive Mode

Kaf has great interactive features. Try producing with pretty formatting:

```bash
kaf produce users-v2 --key
```

In another terminal, consume interactively:
```bash
kaf consume users-v2 --follow
```

Watch messages appear in real-time with nice formatting!

### Task 13: Configuration Contexts

Kaf supports multiple cluster configurations:

```bash
# Add another cluster (example)
kaf config add-cluster production \
  --brokers prod-broker1:9092,prod-broker2:9092

# List clusters
kaf config ls

# Switch between clusters
kaf config use-cluster local
kaf config use-cluster production
```

View all configuration:
```bash
kaf config get-cluster
```

## Verification

You've successfully completed this exercise when you can:
- ✅ Install and configure Kaf
- ✅ List and describe topics with cleaner output
- ✅ Produce and consume messages more easily
- ✅ Work with consumer groups
- ✅ Manage offsets
- ✅ Switch between cluster configurations

## Key Advantages of Kaf

1. **Simpler Syntax**: Less verbose commands
2. **Better UX**: Colored output, tables, readable formats
3. **Configuration Management**: Easy cluster switching
4. **Interactive Features**: Real-time following, better prompts
5. **Sensible Defaults**: Less flags needed for common operations
6. **Schema Registry Support**: Built-in Avro/Protobuf support (covered in later exercises)

## Command Comparison

| Operation | Kafka Native | Kaf |
|-----------|-------------|-----|
| List topics | `kafka-topics --bootstrap-server localhost:9092 --list` | `kaf topics` |
| Create topic | `kafka-topics --bootstrap-server localhost:9092 --create --topic test --partitions 3` | `kaf topic create test -p 3` |
| Produce | `kafka-console-producer --bootstrap-server localhost:9092 --topic test` | `kaf produce test` |
| Consume | `kafka-console-consumer --bootstrap-server localhost:9092 --topic test --from-beginning` | `kaf consume test --offset oldest` |
| List groups | `kafka-consumer-groups --bootstrap-server localhost:9092 --list` | `kaf groups` |

## Tips

- Use `kaf <command> --help` for detailed help
- Tab completion is available for bash/zsh
- Use `--raw` flag for machine-readable output
- Use `--follow` to tail topics in real-time

## Cleanup

Keep the broker running, or stop it with:
```bash
docker compose down
```

## Next Steps

Continue to [Exercise 09: Consumer Groups](../09-consumer-groups/) to dive deeper into consumer group management with Kaf.
