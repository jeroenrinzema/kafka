# Exercise 03: Producer Basics

## Learning Objectives

- Understand the role of producers in Kafka
- Produce messages using the console producer
- Send messages with and without keys
- Understand message serialization

## Background

Producers are applications that publish (write) data to Kafka topics. In this exercise, you'll use the `kafka-console-producer` tool to manually send messages to topics.

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Connect to the broker:
```bash
docker exec -it broker bash
cd /opt/kafka/bin
```

3. Create a topic for this exercise:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic user-events \
  --partitions 3
```

## Tasks

### Task 1: Produce Simple Messages

Start the console producer:
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic user-events
```

Type some messages (press Enter after each):
```
User logged in
User viewed product
User added to cart
User completed purchase
```

Press `Ctrl+C` to exit the producer.

**Note**: Each line you type becomes a separate message.

### Task 2: Produce Messages with Keys

Messages can have keys, which determine which partition they go to. Start a producer with key support:

```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic user-events \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Send messages in the format `key:value`:
```
user123:logged in
user456:logged in
user123:viewed product page
user789:logged in
user123:added item to cart
user456:completed purchase
```

**Note**: All messages with the same key will go to the same partition.

### Task 3: Produce JSON Messages

Create a topic for structured data:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic user-profiles
```

Start the producer:
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic user-profiles
```

Send JSON messages:
```json
{"userId":"123","name":"Alice","country":"US"}
{"userId":"456","name":"Bob","country":"UK"}
{"userId":"789","name":"Charlie","country":"CA"}
```

### Task 4: Produce from a File

Create a sample data file:
```bash
cat > /tmp/messages.txt << EOF
First message from file
Second message from file
Third message from file
Fourth message from file
Fifth message from file
EOF
```

Produce all messages from the file:
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic user-events < /tmp/messages.txt
```

### Task 5: Understanding Producer Acknowledgments

Produce with different acknowledgment settings. First, with no acknowledgment (fire and forget):

```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic user-events \
  --producer-property acks=0
```

Then with leader acknowledgment:
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic user-events \
  --producer-property acks=1
```

**Note**: `acks=0` is fastest but least safe, `acks=1` waits for leader, `acks=all` waits for all replicas.

## Verification

You've successfully completed this exercise when you can:
- ✅ Send simple text messages to a topic
- ✅ Send messages with keys
- ✅ Send JSON-formatted messages
- ✅ Send messages from a file
- ✅ Understand different acknowledgment modes

## Key Concepts

- **Producer**: Application that writes data to Kafka
- **Message Key**: Optional identifier that determines partitioning
- **Serialization**: Converting data to bytes for storage
- **Acknowledgments (acks)**: Confirmation level required before considering a write successful

## Cleanup

Keep the broker running for the next exercise, or stop it with:
```bash
docker compose down
```

## Next Steps

Continue to [Exercise 04: Consumer Basics](../04-consumer-basics/)
