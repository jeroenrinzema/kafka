# Exercise 5.02: SASL/SCRAM Authentication

## Learning Objectives

- Understand the difference between SASL/PLAIN and SASL/SCRAM
- Configure Kafka with SCRAM-SHA-256/512 authentication
- Create and manage SCRAM credentials dynamically
- Connect clients using SCRAM authentication
- Combine SCRAM with ACLs for secure access control

## Background

In exercise 5.01, we used SASL/PLAIN authentication where credentials are defined in a static configuration file. While simple, SASL/PLAIN has limitations:

- Passwords are sent in cleartext (requires SSL for security)
- Credentials are stored in plain text in configuration files
- Adding/removing users requires broker restart

**SASL/SCRAM** (Salted Challenge Response Authentication Mechanism) addresses these issues:

- Passwords are never sent over the wire
- Uses salted, iterated hashing for credential storage
- Credentials can be managed dynamically (no broker restart needed)
- Supports SCRAM-SHA-256 and SCRAM-SHA-512 algorithms

### How SCRAM Works

1. Server stores: `salt`, `iteration-count`, `StoredKey`, `ServerKey`
2. Client proves it knows the password without revealing it
3. Server proves it knows the credentials without revealing them
4. Both parties verify each other (mutual authentication)

## Prerequisites

- Completed Exercise 5.01 (ACLs with SASL/PLAIN)
- Docker and Docker Compose installed
- Basic understanding of Kafka authentication

## Setup

Start Kafka with SCRAM enabled:
```bash
docker compose up -d
```

Wait about 15 seconds for Kafka to fully start. The startup script automatically creates SCRAM credentials for the `admin` user during cluster initialization.

Verify Kafka is running:
```bash
docker logs kafka 2>&1 | grep "Kafka Server started"
```

## Tasks

### Task 1: Set Up Admin Credentials

Create admin credentials file:
```bash
cat > admin.properties << EOF
security.protocol=SASL_PLAINTEXT
sasl.mechanism=SCRAM-SHA-256
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="admin" password="admin-secret";
EOF
```

Copy admin properties to the container:
```bash
docker cp admin.properties kafka:/tmp/admin.properties
```

Test admin connection:
```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

You should see an empty list (no topics yet). This confirms SCRAM authentication works! ✅

### Task 2: Create SCRAM Users

Unlike SASL/PLAIN, SCRAM users are created dynamically using the `kafka-configs.sh` tool.

Create a producer user with SCRAM-SHA-256:
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'SCRAM-SHA-256=[iterations=8192,password=producer-secret]' \
  --entity-type users \
  --entity-name producer-app \
  --command-config /tmp/admin.properties
```

Add SCRAM-SHA-512 for the same user (run as separate command):
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'SCRAM-SHA-512=[iterations=8192,password=producer-secret]' \
  --entity-type users \
  --entity-name producer-app \
  --command-config /tmp/admin.properties
```

Create a consumer user:
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'SCRAM-SHA-256=[iterations=8192,password=consumer-secret]' \
  --entity-type users \
  --entity-name consumer-app \
  --command-config /tmp/admin.properties

docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'SCRAM-SHA-512=[iterations=8192,password=consumer-secret]' \
  --entity-type users \
  --entity-name consumer-app \
  --command-config /tmp/admin.properties
```

### Task 3: Verify SCRAM Credentials

List all SCRAM credentials:
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type users \
  --command-config /tmp/admin.properties
```

You should see the SCRAM credentials (hashed, not the actual passwords) for all three users.

Describe a specific user:
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type users \
  --entity-name producer-app \
  --command-config /tmp/admin.properties
```

### Task 4: Create Topics and ACLs

Create a test topic:
```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

Grant producer permissions:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:producer-app \
  --operation Write \
  --operation Describe \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties

docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:producer-app \
  --operation IdempotentWrite \
  --cluster \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

Grant consumer permissions:
```bash
docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:consumer-app \
  --operation Read \
  --operation Describe \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties

docker exec kafka /opt/kafka/bin/kafka-acls.sh \
  --add \
  --allow-principal User:consumer-app \
  --operation Read \
  --group scram-consumer-group \
  --bootstrap-server localhost:9092 \
  --command-config /tmp/admin.properties
```

### Task 5: Test SCRAM Authentication

Test producing with the producer user:
```bash
echo '{"order": 1, "product": "laptop", "secure": true}' | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=SCRAM-SHA-256 \
  --producer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="producer-app" password="producer-secret";'
```

Produce another message:
```bash
echo '{"order": 2, "product": "keyboard", "secure": true}' | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=SCRAM-SHA-256 \
  --producer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="producer-app" password="producer-secret";'
```

### Task 6: Test Wrong Password (Authentication Failure)

Try with wrong password (should FAIL):
```bash
echo "This should fail" | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=SCRAM-SHA-256 \
  --producer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="producer-app" password="wrong-password";'
```

You should see: `Authentication failed during authentication due to invalid credentials with SASL mechanism SCRAM-SHA-256` ✅

### Task 7: Test Consuming with SCRAM

Consume messages:
```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic scram-orders \
  --from-beginning \
  --bootstrap-server localhost:9092 \
  --group scram-consumer-group \
  --consumer-property security.protocol=SASL_PLAINTEXT \
  --consumer-property sasl.mechanism=SCRAM-SHA-256 \
  --consumer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="consumer-app" password="consumer-secret";' \
  --max-messages 2
```

You should see the messages! ✅

### Task 8: Test SCRAM-SHA-512

SCRAM-SHA-512 provides stronger hashing. Test it:
```bash
echo '{"order": 3, "product": "monitor", "hash": "SHA-512"}' | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=SCRAM-SHA-512 \
  --producer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="producer-app" password="producer-secret";'
```

Consume with SHA-512:
```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic scram-orders \
  --from-beginning \
  --bootstrap-server localhost:9092 \
  --group scram-consumer-group-512 \
  --consumer-property security.protocol=SASL_PLAINTEXT \
  --consumer-property sasl.mechanism=SCRAM-SHA-512 \
  --consumer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="consumer-app" password="consumer-secret";' \
  --max-messages 3
```

### Task 9: Update a User's Password

One of SCRAM's advantages is dynamic credential updates. Let's change the producer's password:
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'SCRAM-SHA-256=[iterations=8192,password=new-producer-secret]' \
  --entity-type users \
  --entity-name producer-app \
  --command-config /tmp/admin.properties

docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --add-config 'SCRAM-SHA-512=[iterations=8192,password=new-producer-secret]' \
  --entity-type users \
  --entity-name producer-app \
  --command-config /tmp/admin.properties
```

Test with old password (should FAIL):
```bash
echo "Old password test" | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=SCRAM-SHA-256 \
  --producer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="producer-app" password="producer-secret";'
```

Test with new password (should work):
```bash
echo '{"order": 4, "product": "mouse", "password": "updated"}' | docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
  --topic scram-orders \
  --bootstrap-server localhost:9092 \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=SCRAM-SHA-256 \
  --producer-property 'sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="producer-app" password="new-producer-secret";'
```

Password changed without restarting the broker! ✅

### Task 10: Delete a User

Remove the consumer user's SCRAM credentials (must be done separately for each mechanism):
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --delete-config 'SCRAM-SHA-256' \
  --entity-type users \
  --entity-name consumer-app \
  --command-config /tmp/admin.properties

docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --alter \
  --delete-config 'SCRAM-SHA-512' \
  --entity-type users \
  --entity-name consumer-app \
  --command-config /tmp/admin.properties
```

Verify the user's credentials are gone:
```bash
docker exec kafka /opt/kafka/bin/kafka-configs.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --entity-type users \
  --entity-name consumer-app \
  --command-config /tmp/admin.properties
```

The user can no longer authenticate! This is useful for immediate revocation.

## Verification

You've successfully completed this exercise when you can:
- ✅ Create SCRAM users dynamically
- ✅ Authenticate with SCRAM-SHA-256 and SCRAM-SHA-512
- ✅ Detect failed authentication with wrong passwords
- ✅ Combine SCRAM with ACLs
- ✅ Update user passwords without restarting the broker
- ✅ Delete user credentials for immediate revocation

## Key Concepts

### SASL/PLAIN vs SASL/SCRAM

| Feature | SASL/PLAIN | SASL/SCRAM |
|---------|------------|------------|
| Password transmission | Cleartext (needs SSL) | Challenge-response (never sent) |
| Credential storage | Plain text in JAAS | Hashed in Kafka metadata |
| Dynamic user management | No (requires restart) | Yes |
| Hash algorithms | N/A | SHA-256, SHA-512 |
| Performance | Slightly faster | Slightly slower |
| Security | Lower (without SSL) | Higher |

### SCRAM Iterations

The `iterations` parameter controls security vs. performance:
- Higher iterations = more secure but slower authentication
- Recommended: 4096 minimum, 8192 for sensitive systems
- Default: 4096

### KRaft Mode and SCRAM Bootstrap

In KRaft mode (Kafka without ZooKeeper), SCRAM credentials are stored in the metadata log. The initial admin user must be created during cluster formatting using `kafka-storage.sh format --add-scram`. This exercise handles this automatically via the startup script.

### When to Use SCRAM

- When you need dynamic user management
- When you can't use SSL (though SSL + SCRAM is best)
- When you need better password security
- For environments with frequent credential rotation

## Cleanup

Stop Kafka:
```bash
docker compose down
```

Remove credentials files:
```bash
rm -f admin.properties
```

## Troubleshooting

**"Authentication failed: Invalid username or password"**
- Verify the username exists: use `--describe --entity-type users`
- Check the password matches what was set with `--add-config`
- Ensure SCRAM mechanism matches (SHA-256 vs SHA-512)

**"No SCRAM credentials found"**
- Make sure you created credentials with `kafka-configs.sh --alter --add-config`
- Verify Kafka has started with SCRAM enabled

**"SCRAM-SHA-256 not enabled"**
- Check `KAFKA_SASL_ENABLED_MECHANISMS` includes SCRAM-SHA-256/512
- Verify `KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL` is set correctly

**"DuplicateResourceException" when creating credentials**
- SCRAM-SHA-256 and SCRAM-SHA-512 must be added in separate commands
- You cannot add both in the same `--add-config` call

## Best Practices

1. **Use SSL with SCRAM**: For encryption in transit
2. **Use SCRAM-SHA-512**: For stronger hashing when possible
3. **High iteration count**: Use at least 8192 for production
4. **Regular credential rotation**: SCRAM makes this easy
5. **Audit credential changes**: Monitor `kafka-configs.sh` usage
6. **Immediate revocation**: Delete credentials when users leave

## Next Steps

- Exercise 5.03: SSL/TLS Encryption - Add encryption to your SCRAM setup
- Combine SCRAM + SSL for the strongest security
- Implement automated credential rotation