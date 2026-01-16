# Exercise 5.01: Access Control Lists (ACLs)

## Learning Objectives

- Understand Kafka authentication and authorization
- Create ACLs to control topic access
- Grant producer and consumer permissions
- Test access with different users
- List and manage ACLs

## Background

Kafka ACLs (Access Control Lists) let you control who can access your topics. This is important for security in production systems.

**Authentication** = Who you are (username/password)  
**Authorization** = What you can do (ACLs)

This exercise uses SASL/PLAIN authentication (username/password) with three users:
- `admin` / `admin-secret` - Full access (super user)
- `producer-app` / `producer-secret` - Will get write access
- `consumer-app` / `consumer-secret` - Will get read access

## Setup

Start Kafka with security enabled:
```bash
docker compose up -d
```

Wait about 10 seconds for Kafka to start.

Create admin credentials file:
```bash
cat > admin.properties << EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";
EOF
```

## Tasks

### Task 1: Create Topics as Admin

Create test topics:
```bash
docker cp admin.properties kafka:/tmp/admin.properties

docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties

docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic payments \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

List topics to verify:
```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

### Task 2: Try Unauthorized Access

Try to produce without permissions (this will FAIL):
```bash
echo "test message" | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=PLAIN \
  --producer-property sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="producer-app" password="producer-secret";'
```

You should see: `Not authorized to access topics: [orders]`

### Task 3: Grant Producer Permissions

Grant write permission:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:producer-app \
  --operation Write \
  --operation Describe \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

Grant cluster permission (needed for modern producers):
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:producer-app \
  --operation IdempotentWrite \
  --cluster \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

### Task 4: Test Authorized Production

Now try producing messages:
```bash
echo '{"order": 1, "product": "laptop"}' | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=PLAIN \
  --producer-property sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="producer-app" password="producer-secret";'

echo '{"order": 2, "product": "mouse"}' | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=PLAIN \
  --producer-property sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="producer-app" password="producer-secret";'
```

Success! ✅

### Task 5: List ACLs

View all ACLs:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --list \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

View ACLs for a specific topic:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --list \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

### Task 6: Grant Consumer Permissions

Grant read permission on topic:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:consumer-app \
  --operation Read \
  --operation Describe \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

Grant read permission on consumer group:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:consumer-app \
  --operation Read \
  --group my-consumer-group \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

### Task 7: Test Consumer Access

Consume messages:
```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic orders \
  --from-beginning \
  --bootstrap-server localhost:9092 \
  --consumer-property security.protocol=SASL_PLAINTEXT \
  --consumer-property sasl.mechanism=PLAIN \
  --consumer-property sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="consumer-app" password="consumer-secret";' \
  --group my-consumer-group \
  --max-messages 2
```

You should see the messages! ✅

### Task 8: Test Access Restrictions

Try to consume from a different topic (should FAIL):
```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic payments \
  --from-beginning \
  --bootstrap-server localhost:9092 \
  --consumer-property security.protocol=SASL_PLAINTEXT \
  --consumer-property sasl.mechanism=PLAIN \
  --consumer-property sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="consumer-app" password="consumer-secret";' \
  --group my-consumer-group \
  --max-messages 1
```

This demonstrates **least privilege** - users only have access to what they need.

### Task 9: Remove an ACL

Remove the write permission:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --remove \
  --allow-principal User:producer-app \
  --operation Write \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties \
  --force
```

Verify it's gone:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --list \
  --topic orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

## Verification

You've successfully completed this exercise when you can:
- ✅ Authenticate with different users
- ✅ Create ACLs for producers and consumers
- ✅ Test that unauthorized access is blocked
- ✅ Test that authorized access works
- ✅ List and remove ACLs

## Key Concepts

- **SASL/PLAIN**: Simple username/password authentication
- **ACL**: Access Control List - defines who can do what
- **Principal**: The user (e.g., `User:producer-app`)
- **Operations**: Read, Write, Describe, IdempotentWrite, etc.
- **Super User**: User that bypasses all ACLs (admin in this exercise)
- **Least Privilege**: Give users only the permissions they need

## Common Permissions

**Producer needs:**
- Write + Describe on topic
- IdempotentWrite on cluster

**Consumer needs:**
- Read + Describe on topic
- Read on consumer group

## Cleanup

Stop Kafka:
```bash
docker compose down
```

Remove credentials file:
```bash
rm -f admin.properties
```

## Troubleshooting

**"Not authorized to access topics"**
- Check you granted both Write and Describe (for producers)
- Check you granted IdempotentWrite on cluster (for producers)
- Check you granted Read and Describe (for consumers)
- Check you granted Read on consumer group (for consumers)

**"Authentication failed"**
- Verify username and password in .properties file
- Check syntax of sasl.jaas.config (backslash and semicolon)

**ACL not working**
- Wait a few seconds for ACL to take effect
- Verify ACL was created with `--list` command

## Next Steps

In production, consider:
- SASL/SCRAM for better password security
- SSL/TLS for encryption
- More granular ACLs per application
- Regular ACL audits
