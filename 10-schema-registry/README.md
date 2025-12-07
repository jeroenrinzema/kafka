# Exercise 10: Schema Registry

## Learning Objectives

- Understand what Schema Registry is and why it's important
- Learn about schema evolution and compatibility modes
- Register and manage Avro schemas
- Produce and consume messages with schema validation
- Explore schema compatibility rules

## What is Schema Registry?

Schema Registry is a centralized repository for managing and validating schemas for topic message data. It provides:

- **Schema Versioning**: Track schema changes over time
- **Compatibility Checking**: Ensure new schemas don't break existing consumers
- **Schema Evolution**: Support for backward, forward, and full compatibility
- **Centralized Management**: Single source of truth for message structure

## Prerequisites

- Completed exercises 1-9
- Docker and Docker Compose installed
- `curl` or similar HTTP client
- Optional: `jq` for JSON formatting

## Tasks

### Task 1: Start the Environment

Start Kafka, Schema Registry, and Kafka UI:

```bash
docker compose up -d
```

Wait a few seconds for services to be ready. Verify Schema Registry is running:

```bash
curl http://localhost:8081/
```

You should see: `[]`

### Task 2: Register Your First Schema

Create an Avro schema for a user profile. View the schema file `schemas/user-v1.avsc`:

```bash
cat schemas/user-v1.avsc
```

This shows our initial user schema with id, username, and email fields.

Register it with Schema Registry using the file:

```bash
SCHEMA=$(cat schemas/user-v1.avsc | jq -c . | jq -R .)
curl -X POST http://localhost:8081/subjects/users-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}"
```

**Expected Output**: `{"id":1}` - This is the schema ID.

### Task 3: List All Subjects

View all registered schema subjects:

```bash
curl http://localhost:8081/subjects
```

**Expected Output**: `["users-value"]`

### Task 4: Retrieve Schema Details

Get the schema we just registered:

```bash
curl http://localhost:8081/subjects/users-value/versions/1
```

You'll see the schema ID, version, and the full schema definition.
### Task 5: Schema Evolution - Add a Field

Let's evolve our schema by adding an optional field (with a default value for backward compatibility).

View the new schema `schemas/user-v2.avsc` which adds a `created_at` field:

```bash
cat schemas/user-v2.avsc
```

Register the new version:

```bash
SCHEMA=$(cat schemas/user-v2.avsc | jq -c . | jq -R .)
curl -X POST http://localhost:8081/subjects/users-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}"
```

**Expected Output**: `{"id":2}` - New version!

### Task 6: Test Schema Compatibility

Check if a new schema is compatible before registering it. View `schemas/user-v3.avsc` which adds another nullable field:

```bash
cat schemas/user-v3.avsc
```

Test compatibility:

```bash
SCHEMA=$(cat schemas/user-v3.avsc | jq -c . | jq -R .)
curl -X POST http://localhost:8081/compatibility/subjects/users-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}"
```

**Expected Output**: `{"is_compatible":true}`EMA=$(cat schemas/user-v3.avsc | jq -c . | jq -R .)
curl -X POST http://localhost:8081/compatibility/subjects/users-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}"
```

**Expected Output**: `{"is_compatible":true}`
```

### Task 7: Try an Incompatible Schema

Try to register a schema that removes a required field (breaking change). The schema `schemas/user-incompatible.avsc` removes the `email` field:

```json
{
  "type": "record",
  "name": "User",
  "namespace": "com.example",
  "fields": [
    {"name": "id", "type": "int"},
    {"name": "username", "type": "string"}
  ]
}
```

Test this incompatible schema:

```bash
SCHEMA=$(cat schemas/user-incompatible.avsc | jq -c . | jq -R .)
curl -X POST http://localhost:8081/compatibility/subjects/users-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}"
```

**Expected Output**: `{"is_compatible":false}`

This protects you from breaking existing consumers!

### Task 7: Try an Incompatible Schema

### Task 7: Try an Incompatible Schema

Try to register a schema that removes a required field (breaking change). View `schemas/user-incompatible.avsc` which removes the `email` field:

```bash
cat schemas/user-incompatible.avsc
```

Test this incompatible schema:

```bash
SCHEMA=$(cat schemas/user-incompatible.avsc | jq -c . | jq -R .)
curl -X POST http://localhost:8081/compatibility/subjects/users-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}"
```

**Expected Output**: `{"is_compatible":false}`

This protects you from breaking existing consumers!

### Task 8: View Compatibility Mode

Check the current compatibility mode:

```bash
curl http://localhost:8081/config/users-value
```

**Default**: `{"compatibilityLevel":"BACKWARD"}`
View all registered subjects:

```bash
curl http://localhost:8081/subjects
```

You should see both `users-value` and `products-value`.

### Task 12: Explore with Kafka UI

Open http://localhost:8080 in your browser.

1. Navigate to "Schema Registry" in the left menu
2. Explore the registered schemas
3. View different versions
4. Check compatibility settings

### Task 13: Produce Messages with Schema (Optional)

If you have `kaf` installed with Avro support, you can produce messages:

```bash
# Create the topic first
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --create \
### Task 10: Create a Product Schema

Register a different schema for products. View `schemas/product-v1.avsc`:

```bash
cat schemas/product-v1.avsc
```

Register it:

```bash
SCHEMA=$(cat schemas/product-v1.avsc | jq -c . | jq -R .)
curl -X POST http://localhost:8081/subjects/products-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}"
```
### Task 14: Delete a Schema Version (Soft Delete)

Soft delete version 1 of the users schema:

```bash
curl -X DELETE http://localhost:8081/subjects/users-value/versions/1
```

List versions to see it's still there (soft deleted):

```bash
curl http://localhost:8081/subjects/users-value/versions
```

### Task 15: Clean Up

Stop and remove all containers:

```bash
docker compose down
```

## Key Concepts

### Compatibility Modes

- **BACKWARD** (default): New schema can read data written with old schema
  - Can delete fields or add optional fields
  - Consumers can upgrade first

- **FORWARD**: Old schema can read data written with new schema
  - Can add fields or delete optional fields
  - Producers can upgrade first

- **FULL**: Both backward and forward compatible
  - Most restrictive but safest
  - Can only add/remove optional fields

- **NONE**: No compatibility checking
  - Use with caution!

### Schema Subject Naming

- `<topic-name>-key`: Schema for message keys
- `<topic-name>-value`: Schema for message values

### Benefits of Schema Registry

1. **Data Quality**: Ensures data consistency across producers and consumers
2. **Evolution**: Safely evolve data structures over time
3. **Documentation**: Schemas serve as living documentation
4. **Efficiency**: Schemas stored centrally, only schema ID sent with messages
5. **Validation**: Prevents invalid data from entering the system

## Troubleshooting

**Schema Registry not responding:**
```bash
docker compose logs schema-registry
```

**Reset everything:**
```bash
docker compose down
docker compose up -d
```

**Check connectivity:**
```bash
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092
```

## Additional Exploration

1. Try creating schemas with different Avro types (arrays, maps, enums)
2. Experiment with different compatibility modes
3. Practice schema evolution patterns
4. Explore the Schema Registry REST API documentation

## Resources

- [Confluent Schema Registry Documentation](https://docs.confluent.io/platform/current/schema-registry/index.html)
- [Avro Specification](https://avro.apache.org/docs/current/spec.html)
- [Schema Evolution Best Practices](https://docs.confluent.io/platform/current/schema-registry/avro.html)
