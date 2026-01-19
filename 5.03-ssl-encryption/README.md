# Exercise 5.03: SSL/TLS Encryption

## Learning Objectives

- Understand SSL/TLS encryption in Kafka
- Generate certificates and keystores for secure communication
- Configure Kafka brokers with SSL
- Connect producers and consumers using SSL
- Enforce SSL-only connections
- Understand mutual TLS (mTLS) authentication

## Background

SSL/TLS provides encryption for data in transit between Kafka clients and brokers, and between brokers themselves. This is essential for:

- **Encryption**: Protecting data from eavesdropping
- **Authentication**: Verifying the identity of brokers (and optionally clients)
- **Integrity**: Ensuring data hasn't been tampered with

### Key Concepts

- **Certificate Authority (CA)**: Issues and signs certificates
- **Keystore**: Contains the broker's private key and certificate
- **Truststore**: Contains trusted CA certificates
- **One-way SSL**: Only the broker is authenticated (client trusts broker)
- **Two-way SSL (mTLS)**: Both broker and client are authenticated

## Setup

This exercise includes a script to generate all necessary certificates. Start by generating them:

```bash
chmod +x generate-certs.sh
./generate-certs.sh
```

This creates:
- `secrets/ca-cert` and `secrets/ca-key` - Certificate Authority
- `secrets/kafka.keystore.jks` - Broker keystore with private key
- `secrets/kafka.truststore.jks` - Broker truststore with CA cert
- `secrets/client.keystore.jks` - Client keystore (for mTLS)
- `secrets/client.truststore.jks` - Client truststore with CA cert
- `secrets/ca-cert.pem` - CA certificate in PEM format (for Go clients)

Start Kafka with SSL enabled:
```bash
docker compose up -d
```

Wait about 15 seconds for Kafka to start and initialize SSL.

## Tasks

### Task 1: Verify SSL Configuration

Check that Kafka is listening on SSL port:
```bash
docker compose logs kafka | grep -i ssl
```

You should see logs indicating SSL is configured.

Check the advertised listeners:
```bash
docker exec kafka /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server localhost:9093 \
  --command-config /tmp/client-ssl.properties
```

### Task 2: Create a Topic with SSL

The client SSL configuration is already mounted in the container. Create a test topic:

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic secure-orders \
  --bootstrap-server localhost:9093 \
  --replication-factor 1 \
  --partitions 3 \
  --command-config /tmp/client-ssl.properties
```

List topics to verify:
```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9093 \
  --command-config /tmp/client-ssl.properties
```

### Task 3: Test SSL with Console Producer/Consumer

Produce messages over SSL:
```bash
echo '{"orderId": "1001", "amount": 99.99, "customer": "alice@example.com"}' | \
  docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
    --topic secure-orders \
    --bootstrap-server localhost:9093 \
    --producer.config /tmp/client-ssl.properties

echo '{"orderId": "1002", "amount": 149.50, "customer": "bob@example.com"}' | \
  docker exec -i kafka /opt/kafka/bin/kafka-console-producer.sh \
    --topic secure-orders \
    --bootstrap-server localhost:9093 \
    --producer.config /tmp/client-ssl.properties
```

Consume messages over SSL:
```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic secure-orders \
  --from-beginning \
  --bootstrap-server localhost:9093 \
  --consumer.config /tmp/client-ssl.properties \
  --max-messages 2
```

### Task 4: Verify Plain Connection is Rejected

Try to connect without SSL to the SSL port (this should FAIL):
```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9093
```

You should see an SSL handshake error because the client is not configured for SSL.

### Task 5: Test with Go Producer (Optional)

If you have Go installed, the Go producer demonstrates programmatic SSL configuration:

```bash
cd producer
go run main.go
```

This will:
1. Load the CA certificate
2. Configure TLS settings
3. Connect to Kafka over SSL
4. Produce messages to the `secure-orders` topic

Verify messages were received:
```bash
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic secure-orders \
  --from-beginning \
  --max-messages 10 \
  --bootstrap-server localhost:9093 \
  --consumer.config /tmp/client-ssl.properties
```

### Task 6: Test with Go Consumer (Optional)

The Go consumer demonstrates SSL configuration for consuming:

```bash
cd consumer
go run main.go
```

This will:
1. Load the CA certificate
2. Configure TLS settings
3. Connect to Kafka over SSL
4. Consume messages from the `secure-orders` topic

Let it run for a few seconds to see messages, then press Ctrl+C.

### Task 7: Enable Mutual TLS (mTLS) - Advanced

For mutual TLS, the broker also authenticates the client using certificates.

Stop the current setup:
```bash
docker compose down
```

Edit `docker-compose.yaml` and change `KAFKA_SSL_CLIENT_AUTH` from `none` to `required`:
```yaml
KAFKA_SSL_CLIENT_AUTH: required
```

Start with mTLS enabled:
```bash
docker compose up -d
```

Now clients MUST present a valid certificate. Test with the client certificate:
```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9093 \
  --command-config /tmp/client-ssl-mtls.properties
```

You can also verify that the regular SSL config (without client cert) is now rejected:
```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server localhost:9093 \
  --command-config /tmp/client-ssl.properties
```

This should fail with an SSL authentication error because no client certificate was provided.

## Key Concepts Learned

### SSL/TLS Encryption
- Encrypts all data in transit
- Prevents eavesdropping and man-in-the-middle attacks
- Adds some latency overhead (usually acceptable)

### Certificate Management
- CA certificate must be trusted by clients
- Keystores contain private keys (keep secure!)
- Truststores contain public certificates
- Certificates should be rotated periodically

### One-way vs Two-way SSL
- **One-way**: Client verifies broker identity (most common)
- **Two-way (mTLS)**: Both client and broker verify each other
- mTLS provides stronger security but requires certificate management for all clients

### Performance Considerations
- SSL adds CPU overhead for encryption/decryption
- Impact is usually 5-15% for moderate throughput
- Use hardware acceleration where available
- Consider SSL only for external clients, plaintext for inter-broker

## Common Issues

### Certificate Verification Failed
- Ensure the CA certificate is in the client's truststore
- Check that the broker's certificate hostname matches the connection hostname
- Verify certificate hasn't expired: `keytool -list -v -keystore kafka.keystore.jks`

### Connection Timeout
- Ensure the SSL port (9093) is accessible
- Check firewall rules
- Verify the broker is listening on SSL: `docker compose logs kafka | grep SSL`

### Wrong Password
- The keystore and truststore passwords must match what's configured
- Default in this exercise: `kafka-secret`

## Cleanup

Stop and remove containers:
```bash
docker compose down
```

To clean up generated certificates:
```bash
rm -rf secrets/
```

## Best Practices

1. **Use Strong Passwords**: Replace default passwords in production
2. **Rotate Certificates**: Set up a process to rotate certificates before expiry
3. **Secure Private Keys**: Limit access to keystore files
4. **Monitor Certificate Expiry**: Alert before certificates expire
5. **Use mTLS for Sensitive Data**: When you need client authentication
6. **Separate Internal/External**: Use SSL for external, consider plaintext internally
7. **Test Thoroughly**: Verify SSL is actually being used (network captures, logs)

## Additional Resources

- [Kafka Security Documentation](https://kafka.apache.org/documentation/#security)
- [SSL/TLS Encryption](https://kafka.apache.org/documentation/#security_ssl)
- [Java Keytool Reference](https://docs.oracle.com/javase/8/docs/technotes/tools/unix/keytool.html)

## Next Steps

Try these challenges:
1. Generate new certificates with different key sizes (2048 vs 4096 bits)
2. Set up a three-broker cluster with SSL inter-broker communication
3. Implement certificate rotation without downtime
4. Monitor SSL connection metrics using JMX
5. Combine SSL with SASL for encryption + authentication
