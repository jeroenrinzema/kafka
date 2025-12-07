# Exercise 04: Consumer Basics

## Learning Objectives

- Understand the role of consumers in Kafka
- Consume messages using the console consumer
- Read messages from the beginning vs. latest
- Display message keys and metadata
- Understand consumer groups

## Background

Consumers are applications that read data from Kafka topics. They subscribe to one or more topics and process the stream of records. In this exercise, you'll use the `kafka-console-consumer` tool to read messages.

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Open two terminal windows and connect to the broker in both:
```bash
# Terminal 1 (Producer)
docker exec -it broker bash
cd /opt/kafka/bin

# Terminal 2 (Consumer)
docker exec -it broker bash
cd /opt/kafka/bin
```

3. Create a topic and add some initial data:
```bash
# In Terminal 1
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic events \
  --partitions 3

./kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Then type:
```
user1:User logged in
user2:User logged in
user1:User viewed homepage
user3:User logged in
user1:User searched for "kafka"
user2:User viewed product
```

Press `Ctrl+C` when done.

## Tasks

### Task 1: Consume from the Latest Offset

In Terminal 2, start a consumer that reads new messages only:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events
```

**Note**: By default, the consumer starts reading from the latest offset, so it only sees new messages that are produced AFTER the consumer starts.

Now in Terminal 1, produce new messages:
```bash
./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic events
```

Type some messages and observe them appearing in Terminal 2 in real-time.

### Task 2: Consume from the Beginning

Stop the consumer (`Ctrl+C`) and restart it with the `--from-beginning` flag:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --from-beginning
```

**Question**: What messages do you see now?

### Task 3: Display Keys and Metadata

Stop the consumer and restart it to display keys and other metadata:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --from-beginning \
  --property print.key=true \
  --property print.value=true \
  --property print.partition=true \
  --property print.offset=true \
  --property print.timestamp=true \
  --property key.separator=" | "
```

**Observe**:
- Which partition each message is in
- The offset of each message
- The timestamp
- The key and value

**Question**: Do messages with the same key go to the same partition?

### Task 4: Consume from a Specific Partition

Stop the consumer and consume from only partition 0:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --partition 0 \
  --from-beginning
```

Try partitions 1 and 2 as well. 

**Question**: How are messages distributed across partitions?

### Task 5: Format Output

You can format the output for better readability. Try consuming JSON messages:

First, create and populate a topic with JSON:
```bash
# In Terminal 1
./kafka-topics.sh --bootstrap-server localhost:9092 --create --topic users

./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic users
```

Type some JSON:
```json
{"id":1,"name":"Alice","email":"alice@example.com"}
{"id":2,"name":"Bob","email":"bob@example.com"}
{"id":3,"name":"Charlie","email":"charlie@example.com"}
```

Now consume with formatting:
```bash
# In Terminal 2
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic users \
  --from-beginning \
  --property print.key=false \
  --property print.value=true
```

### Task 6: Use a Consumer Group

Consumer groups allow multiple consumers to share the workload. Start a consumer with a group ID:

```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --group my-app-group \
  --from-beginning
```

Once it finishes reading, stop it (`Ctrl+C`) and restart the same command.

**Question**: Does it read the messages again? Why or why not?

### Task 7: Consumer Group Details

Check the status of your consumer group:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group my-app-group \
  --describe
```

**Observe**:
- Current offset for each partition
- Log end offset
- Lag (difference between log end and current offset)

List all consumer groups:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list
```

### Task 8: Reset Consumer Group Offsets

Reset the consumer group to read from the beginning again:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group my-app-group \
  --topic events \
  --reset-offsets \
  --to-earliest \
  --execute
```

Now start the consumer again:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --group my-app-group
```

**Question**: Does it read all messages again?

## Verification

You've successfully completed this exercise when you can:
- ✅ Consume messages from the latest offset
- ✅ Consume messages from the beginning
- ✅ Display message keys, partitions, and offsets
- ✅ Consume from a specific partition
- ✅ Use consumer groups
- ✅ Check consumer group status and lag
- ✅ Reset consumer group offsets

## Key Concepts

- **Consumer**: Application that reads data from Kafka
- **Consumer Group**: Group of consumers sharing the workload of reading a topic
- **Offset**: Position of a consumer in a partition
- **Lag**: Number of messages a consumer is behind the latest message
- **Deserialization**: Converting bytes back to readable data

## Cleanup

Keep the broker running for the next exercise, or stop it with:
```bash
docker compose down
```

## Next Steps

Continue to [Exercise 05: Topic Partitions](../05-topic-partitions/)
