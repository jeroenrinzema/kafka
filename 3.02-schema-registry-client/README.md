# Exercise 11: Producing and Consuming with Schema Registry

## Learning Objectives

- Implement a producer that serializes messages using Avro and Schema Registry
- Implement a consumer that deserializes Avro messages
- Understand automatic schema registration
- Handle schema evolution in real applications
- Use the Confluent Kafka Go client with Schema Registry integration

## What You'll Build

In this exercise, you'll create:
1. **Producer**: Sends user data to Kafka with Avro serialization
2. **Consumer**: Reads and deserializes user data from Kafka
3. Both applications use Schema Registry for schema management

## Prerequisites

- Completed exercises 1-10
- Docker and Docker Compose installed
- Go 1.21 or later installed
- Basic understanding of Avro schemas

## Architecture

```
Producer App → Kafka Topic (users)
    ↓              ↓
Schema Registry ← Consumer App
```

The producer:
1. Reads the Avro schema from file
2. Serializes user data to Avro format
3. Schema Registry automatically handles schema registration
4. Sends binary Avro data to Kafka

The consumer:
1. Receives binary Avro data from Kafka
2. Extracts schema ID from message
3. Fetches schema from Schema Registry
4. Deserializes message using the schema

## Tasks

### Task 1: Start the Environment

Start Kafka, Schema Registry, and Kafka UI:

```bash
docker compose up -d
```

Wait a few seconds for all services to be ready.

Verify Schema Registry is running:

```bash
curl http://localhost:8081/
```

### Task 2: Examine the Schema

Look at the user schema we'll be using:

```bash
cat schemas/user.avsc
```

This defines the structure for our user messages with fields: id, username, email, and created_at.

### Task 3: Create the Kafka Topic

Create the topic where we'll send user data:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --create \
  --topic users \
  --bootstrap-server localhost:9092 \
  --partitions 3 \
  --replication-factor 1
```

Verify the topic was created:

```bash
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092
```

### Task 4: Understand the Producer Code

Open `producer/main.go` and examine the key components:

1. **Schema Registry Client**: Connects to Schema Registry
2. **Avro Serializer**: Handles Avro encoding with automatic schema registration
3. **User Model**: Represents our domain object
4. **Message Production**: Converts users to Avro and sends to Kafka

The producer automatically registers the schema with Schema Registry on the first message!

### Task 5: Run the Producer

Send user messages to Kafka:

```bash
cd producer
go run main.go
```

You should see output like:
```
Produced user: alice (alice@example.com)
Produced user: bob (bob@example.com)
...
Delivered message to users [partition 0] at offset 0
...
```

The producer sends 5 sample users to the topic.

### Task 6: Verify Schema Registration

Check that the schema was automatically registered:

```bash
curl http://localhost:8081/subjects
```

You should see: `["users-value"]`

View the schema details:

```bash
curl http://localhost:8081/subjects/users-value/versions/1 | python3 -m json.tool
```

### Task 7: View Messages in Kafka UI

Open http://localhost:8080 in your browser.

1. Navigate to "Topics" → "users"
2. Click "Messages"
3. Notice the messages show the Avro-deserialized content
4. Navigate to "Schema Registry" to see the registered schema

### Task 8: Understand the Consumer Code

Open `consumer/main.go` and examine:

1. **Avro Deserializer**: Handles Avro decoding with automatic schema fetching
2. **Message Processing**: Converts Avro data back to Go structs
3. **Graceful Shutdown**: Handles SIGINT/SIGTERM signals

The consumer automatically fetches the schema from Schema Registry based on the schema ID in each message!

### Task 9: Run the Consumer

In a new terminal, consume the messages:

```bash
cd consumer
go run main.go
```

You should see output like:
```
Starting consumer, subscribed to topic: users
Waiting for messages... (Press Ctrl+C to exit)
📨 Message 1 | Partition: 0, Offset: 0
   User ID: 1
   Username: alice
   Email: alice@example.com
   Created: 2024-12-03T10:30:45Z
   --------------------------------------------------
...
```

Keep the consumer running for the next tasks.

### Task 10: Produce More Messages

In another terminal, run the producer again:

```bash
cd producer
go run main.go
```

Watch the consumer terminal - it should automatically receive and deserialize the new messages!

### Task 11: Test Schema Evolution

Let's evolve our schema by adding a new optional field. Create a new schema file:

```bash
cat > schemas/user-v2.avsc << 'EOF'
{
  "type": "record",
  "name": "User",
  "namespace": "com.example",
  "fields": [
    {
      "name": "id",
      "type": "int"
    },
    {
      "name": "username",
      "type": "string"
    },
    {
      "name": "email",
      "type": "string"
    },
    {
      "name": "created_at",
      "type": "long"
    },
    {
      "name": "phone",
      "type": ["null", "string"],
      "default": null
    }
  ]
}
EOF
```

Modify the producer to use this new schema and add a phone field to the User struct and userMap. Then run it again.

The consumer will still work because the new field is optional (backward compatible)!

### Task 12: View Schema Versions

Check all versions of the schema:

```bash
curl http://localhost:8081/subjects/users-value/versions
```

You should see: `[1,2]` (if you completed Task 13)

Compare the versions:

```bash
curl http://localhost:8081/subjects/users-value/versions/1 | python3 -m json.tool
curl http://localhost:8081/subjects/users-value/versions/2 | python3 -m json.tool
```

### Task 13: Monitor with Kafka UI

In Kafka UI (http://localhost:8080):

1. Check the message count in the users topic
2. View the schema versions
3. Inspect individual messages
4. Notice how Kafka UI deserializes Avro messages automatically

### Task 14: Test Error Handling

Try to produce a message with invalid data (e.g., wrong type for a field). The Avro serializer will reject it before sending to Kafka, ensuring data quality!

You can modify the producer code to test this.

### Task 15: Clean Up

Stop the consumer (Ctrl+C in its terminal).

Stop and remove all containers:

```bash
docker compose down
```

## Key Concepts

### Automatic Schema Registration

When the producer serializes the first message:
1. It sends the schema to Schema Registry
2. Schema Registry assigns a schema ID
3. The ID is embedded in each message (magic byte + schema ID + Avro payload)
4. Consumers use this ID to fetch the schema

### Schema Evolution

With Schema Registry, you can:
- Add optional fields (backward compatible)
- Remove optional fields (forward compatible)
- Rename fields with aliases
- Change field types (with caution)

### Wire Format

Each Avro message contains:
```
[0x00] [schema-id] [avro-payload]
 byte    4 bytes      variable
```

This is much more efficient than including the full schema in every message!

### Benefits

1. **Type Safety**: Compile-time checks in your application
2. **Efficiency**: Binary format is compact
3. **Evolution**: Safe schema changes without breaking consumers
4. **Documentation**: Schema serves as contract between services
5. **Validation**: Invalid data is rejected before reaching Kafka

## Troubleshooting

**Producer fails to serialize:**
- Check schema file path and format
- Ensure Schema Registry is running
- Verify network connectivity to Schema Registry

**Consumer can't deserialize:**
- Ensure Schema Registry is accessible
- Check that schema exists for the topic
- Verify the schema ID in messages is valid

**Connection errors:**
```bash
# Check Kafka
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --list --bootstrap-server localhost:9092

# Check Schema Registry
curl http://localhost:8081/subjects

# Check logs
docker compose logs kafka
docker compose logs schema-registry
```

## Additional Exercises

1. **Add More Fields**: Extend the schema with additional user attributes
2. **Multiple Schemas**: Create producer/consumer for a different entity (e.g., Product)
3. **Error Handling**: Implement retry logic for Schema Registry failures
4. **Batch Processing**: Modify producer to send messages in batches
5. **Metrics**: Add monitoring for serialization performance
6. **Schema Validation**: Add custom validation logic before serialization

## Best Practices

1. **Always use optional fields for new additions** (backward compatibility)
2. **Test schema changes in dev/staging first**
3. **Use meaningful field names and add documentation**
4. **Cache Schema Registry clients** (they're thread-safe)
5. **Handle deserialization errors gracefully**
6. **Monitor schema evolution** in production

## Resources

- [Confluent Kafka Go Client](https://github.com/confluentinc/confluent-kafka-go)
- [Avro Specification](https://avro.apache.org/docs/current/spec.html)
- [Schema Registry API](https://docs.confluent.io/platform/current/schema-registry/develop/api.html)
- [Schema Evolution Guide](https://docs.confluent.io/platform/current/schema-registry/avro.html)

## Next Steps

Continue to [Exercise 3.03: Kafka Connect Fundamentals](../3.03-connect-fundamentals/) to learn about integrating external data sources with Kafka Connect.
