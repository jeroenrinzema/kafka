# Exercise 06: Offset Management

## Learning Objectives

- Understand what offsets are and how they work
- Consume from specific offsets
- Consume from specific timestamps
- Reset consumer group offsets
- Understand offset management strategies
- Learn about committed offsets

## Background

Offsets are Kafka's way of tracking position in a partition. Each message in a partition has a unique offset. Consumers track their position using offsets, which enables:
- Resuming from where they left off
- Replaying messages
- Skipping to specific points in time
- Processing guarantees (at-least-once, at-most-once)

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

3. Create a topic and populate it with timestamped data:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic events \
  --partitions 3
```

## Tasks

### Task 1: Populate Topic with Time-Series Data

Create a script to add timestamped messages:
```bash
cat > /tmp/populate.sh << 'EOF'
#!/bin/bash
for i in {1..100}; do
  timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  echo "Event $i at $timestamp"
  sleep 0.1
done
EOF

chmod +x /tmp/populate.sh
/tmp/populate.sh | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic events
```

This creates 100 messages over ~10 seconds.

### Task 2: Understanding Offsets

Consume messages and observe their offsets:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --from-beginning \
  --partition 0 \
  --property print.offset=true \
  --property print.timestamp=true \
  --max-messages 10
```

**Observe**:
- Offsets start at 0
- Each message has a unique offset within its partition
- Timestamps are in milliseconds since epoch

### Task 3: Consume from a Specific Offset

Let's say you want to start reading from offset 20 in partition 0:

```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --partition 0 \
  --offset 20 \
  --property print.offset=true \
  --max-messages 10
```

Try different offsets to see different messages.

### Task 4: Find Messages by Timestamp

First, let's get a timestamp from our data. Consume some messages and note a timestamp:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --partition 0 \
  --from-beginning \
  --property print.timestamp=true \
  --property print.offset=true \
  --max-messages 5
```

Note one of the timestamps (in milliseconds). 

**Note**: You can use the `--to-datetime` flag with consumer group reset commands (shown in Task 7) to seek to a specific timestamp. This is the most practical way to find and consume from a specific time.

### Task 5: Reset Consumer Group to Specific Offset

Create a consumer group and let it consume all messages:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --group analytics-team \
  --from-beginning
```

After it finishes, press `Ctrl+C`.

Check the current offsets:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --describe
```

Now reset to offset 50 for all partitions:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --to-offset 50 \
  --execute
```

Verify the reset:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --describe
```

### Task 6: Reset to Earliest/Latest

Reset the consumer group to the earliest offset:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --to-earliest \
  --execute
```

Reset to the latest (end):
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --to-latest \
  --execute
```

### Task 7: Reset to Timestamp

Let's say you want to replay all events from 2 minutes ago. First, get a timestamp from your data:

```bash
# Look at the timestamps of your messages
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic events \
  --partition 0 \
  --from-beginning \
  --property print.timestamp=true \
  --max-messages 5
```

Note one of the earlier timestamps. Now convert it to ISO8601 format and use it:

```bash
# Example: Use an ISO8601 formatted timestamp
# Format must be: YYYY-MM-DDTHH:MM:SS.sss
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --to-datetime "2025-12-03T10:00:00.000" \
  --execute
```

You can also create a timestamp for a specific time:
```bash
# Get current time in ISO8601 format (macOS/Linux)
CURRENT_TIME=$(date -u +"%Y-%m-%dT%H:%M:%S.000")
echo "Current time: $CURRENT_TIME"

./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --to-datetime "$CURRENT_TIME" \
  --execute
```

**Note**: The timestamp must be in ISO8601 format: `YYYY-MM-DDTHH:MM:SS.sss`

### Task 8: Shift Offsets Forward/Backward

Shift backward by 20 messages:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --shift-by -20 \
  --execute
```

Shift all offsets forward by 10 messages:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --shift-by 10 \
  --execute
```

### Task 9: Reset Specific Partition

Reset only partition 1 to offset 30:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events:1 \
  --reset-offsets \
  --to-offset 30 \
  --execute
```

### Task 10: Export and Import Offsets

Export current offsets to a file:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --describe \
  --members \
  --verbose
```

You can also use a CSV file to set specific offsets:
```bash
cat > /tmp/offsets.csv << EOF
events,0,25
events,1,30
events,2,15
EOF

./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --reset-offsets \
  --from-file /tmp/offsets.csv \
  --execute
```

### Task 11: Dry Run vs Execute

Always test with `--dry-run` before `--execute`:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group analytics-team \
  --topic events \
  --reset-offsets \
  --to-earliest \
  --dry-run
```

This shows what WOULD happen without actually changing offsets.

### Task 12: Monitor Lag

Create a new topic for this demonstration:
```bash
./kafka-topics.sh --bootstrap-server localhost:9092 \
  --create \
  --topic lag-demo \
  --partitions 3
```

Add many messages quickly to the new topic:
```bash
for i in {1..150}; do
  echo "Event $i at $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
done | ./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic lag-demo
```

Create the consumer group by consuming and immediately stopping:
```bash
./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic lag-demo \
  --group lag-monitor \
  --from-beginning \
  --max-messages 1
```

Verify the group exists and see the initial state:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group lag-monitor \
  --describe
```

**Observe**: The LAG column shows ~149 messages (150 total - 1 consumed) waiting to be processed.

In a second terminal, start monitoring the consumer group lag (this will update every second):
```bash
# In Terminal 2
docker exec -it broker bash
cd /opt/kafka/bin

watch -n 1 "./kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group lag-monitor --describe"
```

Now, in the first terminal, create a slow consumer script that processes one message at a time:
```bash
# In Terminal 1
cat > /tmp/slow-consumer.sh << 'EOF'
#!/bin/bash
while true; do
  # Consume exactly 1 message and commit
  ./kafka-console-consumer.sh --bootstrap-server localhost:9092 \
    --topic lag-demo \
    --group lag-monitor \
    --max-messages 1 \
    --timeout-ms 2000 2>/dev/null
  
  # If no message was consumed, exit
  if [ $? -ne 0 ]; then
    break
  fi
  
  # Sleep to simulate slow processing
  sleep 2
done
echo "Finished consuming"
EOF

chmod +x /tmp/slow-consumer.sh
/tmp/slow-consumer.sh
```

**Observe** in Terminal 2: 
- Every 2 seconds, the CURRENT-OFFSET increases by 1
- The LAG decreases one message at a time from ~149 down
- You can clearly see the consumer working through messages slowly
- The LOG-END-OFFSET shows the total messages (150)
- This demonstrates how lag accumulates when consumers are slower than producers

After about 100 seconds (50 messages × 2 seconds), press `Ctrl+C` to stop the slow consumer.

Check the final lag:
```bash
./kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group lag-monitor \
  --describe
```

You should see remaining lag of about 100 messages because you only consumed about 50 messages.

## Verification

You've successfully completed this exercise when you can:
- ✅ Understand offset numbering within partitions
- ✅ Consume from a specific offset
- ✅ Reset consumer groups to different positions
- ✅ Reset to specific timestamps
- ✅ Shift offsets forward and backward
- ✅ Reset individual partitions
- ✅ Monitor consumer lag
- ✅ Use dry-run before executing offset changes

## Key Concepts

- **Offset**: Unique position identifier for a message within a partition
- **Committed Offset**: Last offset a consumer has successfully processed
- **Consumer Lag**: Difference between latest offset and consumer's committed offset
- **Offset Reset**: Changing where a consumer starts reading
- **Timestamp-based Seeking**: Finding offsets by message timestamp

## Offset Management Strategies

1. **Auto-commit** (default): Offsets committed automatically every 5 seconds
   - Pro: Simple, no code needed
   - Con: May lead to message loss or duplicates

2. **Manual commit**: Application controls when to commit
   - Pro: Precise control, can implement exactly-once semantics
   - Con: More complex code

3. **Offset storage**: 
   - Kafka stores offsets in internal `__consumer_offsets` topic
   - Can also store offsets externally (database, etc.)

## Common Use Cases

- **Replay for debugging**: Reset to earlier offset to reprocess messages
- **Data recovery**: Reset to specific timestamp after an incident
- **Testing**: Reset to beginning to test consumer logic
- **Skip bad messages**: Shift forward to skip problematic messages
- **Time-travel**: Process data as it existed at a specific time

## Cleanup

Keep the broker running for the next exercise, or stop it with:
```bash
docker compose down
```

## Next Steps

You've completed the fundamentals! Continue to [Exercise 2.01: Kaf Introduction](../2.01-kaf-introduction/) to learn about a modern CLI tool for Kafka.
