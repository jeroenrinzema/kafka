# Exercise 10.01: Debugging Challenge

## Learning Objectives

- Apply troubleshooting techniques learned in previous exercises
- Identify and fix common Kafka producer issues
- Identify and fix common Kafka consumer issues
- Use Kafka CLI tools to diagnose problems
- Understand proper error handling and configuration

## Background

In this exercise, you'll debug a broken order processing system. The producer and consumer have multiple issues that prevent them from working correctly. Your job is to find and fix all the problems using the knowledge you've gained from previous exercises.

## Scenario

An e-commerce company has a order processing system that isn't working. The producer is supposed to send order events to a Kafka topic, and the consumer should process them. However, developers are reporting that:

1. The producer seems to run but orders aren't appearing in Kafka
2. The consumer starts but doesn't process any orders
3. When orders do appear, they might not be properly partitioned by user

Your task: **Find and fix all the issues!**

## Setup

1. Start the Kafka broker:
```bash
docker compose up -d
```

2. Create the topic:
```bash
docker exec -it broker /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --topic orders \
  --partitions 3 \
  --replication-factor 1
```

3. Run the producer:
```bash
cd producer
go run main.go
```

4. Run the consumer:
```bash
cd consumer
go run main.go
```

5. Observe the behavior - what's working? What's not working?

6. Examine the producer code in `producer/main.go`
7. Examine the consumer code in `consumer/main.go`

## Challenge Tasks

Your task is to debug and fix the order processing system. Use the tools and hints below to investigate and resolve the issues.

### Debugging Tools

**Check Kafka topics:**
```bash
# List all topics
docker exec -it broker /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list

# Describe a specific topic
docker exec -it broker /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic orders

# Consume messages from a topic
docker exec -it broker /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic orders \
  --from-beginning \
  --property print.key=true \
  --property print.partition=true \
  --property key.separator=":"
```

**Check consumer groups:**
```bash
# List all consumer groups
docker exec -it broker /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --list

# Describe a consumer group (check lag, offsets)
docker exec -it broker /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group order-processor
```

### Debugging Hints

**When investigating the producer:**
- Verify the broker address and port
- Check acknowledgment settings for production reliability
- Consider partitioning strategy - what should be the message key?

**When investigating the consumer:**
- Verify topic names match between producer and consumer
- Check the offset reset policy
- Consider what happens if the topic is empty or has old messages
- Think about whether auto-commit is appropriate for your use case

### Verification Steps

Once you've fixed both the producer and consumer:

1. **Create the topic with proper configuration:**
```bash
docker exec -it broker /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --create \
  --topic orders \
  --partitions 3 \
  --replication-factor 1
```

2. **Run the fixed producer:**
```bash
cd producer
go run main.go
```

3. **Verify messages are in Kafka:**
```bash
docker exec -it broker /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic orders \
  --from-beginning \
  --property print.key=true \
  --property key.separator=":"
```

4. **Run the fixed consumer:**
```bash
cd consumer
go run main.go
```

5. **Check that messages are partitioned correctly by user:**
```bash
# Check which partition each message went to
Once you believe you've fixed the issues, verify the system works end-to-end:

1. **Create the topic:**
  --from-beginning \
  --property print.key=true \
  --property print.partition=true \
  --property key.separator=":"
```

2. **Run the producer** and verify messages appear in Kafka
3. **Run the consumer** and verify it processes the messages
4. **Check partitioning** - messages with the same user ID should go to the same partition

### Additional Scenarios to Test

Test your fixed code with these scenarios:

**Scenario A: Lost Messages**
- Set `kgo.RequiredAcks(kgo.NoAck())` in the producer
- Immediately shut down the broker after producing
- Do the messages survive? Why or why not?

**Scenario B: Reprocessing**
- Run the consumer once
- Run it again with the same group ID
- Does it reprocess messages? Why or why not?
- How would you force it to reprocess?

**Scenario C: Consumer Lag**
- Produce 100 messages quickly
- Start a slow consumer (add sleep in processing)
- Check consumer lag:
```bash
docker exec -it broker /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group order-processor
```

## Problems Reference

Here's a summary of the issues you should find (don't look until you've tried!):

<details>
<summary>Click to reveal all problems</summary>

**Producer Problems:**
1. Wrong bootstrap server address (`localhost:9093` instead of `localhost:9092`)
2. `kgo.RequiredAcks(kgo.NoAck())` - no acknowledgment, fire-and-forget (risky for production)
3. Using `order.OrderID` as key instead of `order.UserID` (wrong partitioning)

**Consumer Problems:**
4. Wrong topic name (`order-events` instead of `orders`)
5. `kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd())` - only reads new messages, misses historical data
6. Auto-commit without considering exactly-once semantics for critical operations

</details>

## Verification

You've successfully completed this exercise when you can:
- ✅ Identify all issues in the producer code
- ✅ Identify all issues in the consumer code
- ✅ Fix the producer to reliably send messages
- ✅ Fix the consumer to properly receive and process messages
- ✅ Verify messages are partitioned correctly by user ID
- ✅ Explain the impact of each issue and why your fix works
- ✅ Use Kafka CLI tools to diagnose problems

## Key Debugging Techniques

- **Check connectivity**: Verify broker addresses and ports
- **Inspect topics**: Use `kafka-topics.sh` to list and describe topics
- **Monitor consumer groups**: Use `kafka-consumer-groups.sh` to check lag and offsets
- **Read messages directly**: Use `kafka-console-consumer.sh` to verify topic contents
- **Check partitioning**: Use `print.partition=true` to see message distribution
- **Review logs**: Look for error messages and warnings
- **Test incrementally**: Fix one issue at a time and verify

## Common Pitfalls to Avoid

1. **Wrong broker address**: Verify you're using the correct host and port
2. **Topic name mismatches**: Ensure producer and consumer use the same topic name
3. **Offset management**: Understand when to use `AtStart()` vs `AtEnd()`
4. **Acknowledgment modes**: Choose appropriate acknowledgment for your reliability needs
5. **Partitioning strategy**: Use meaningful keys for proper message distribution
6. **Error handling**: Always check and handle errors appropriately

## Cleanup

Stop the containers:
```bash
docker compose down
```

## Next Steps

Now that you've debugged these issues, you can:
- Review exercises 1.03-1.04 on producer/consumer basics
- Study exercise 1.05 on topic partitions
- Explore exercise 1.06 on offset management
- Check exercise 2.03 on retry mechanisms

## Reflection Questions

1. How would you prevent these issues in a real production system?
2. What monitoring would help you detect these problems quickly?
3. Which issue was hardest to find? Why?
4. What testing strategy would catch these bugs before deployment?
