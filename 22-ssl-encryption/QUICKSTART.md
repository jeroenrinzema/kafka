# Quick Start Guide - SSL Encryption Exercise

## TL;DR - Get Started in 5 Minutes

### 1. Generate Certificates
```bash
cd 22-ssl-encryption
chmod +x generate-certs.sh
./generate-certs.sh
```

### 2. Start Kafka with SSL
```bash
docker compose up -d
sleep 15  # Wait for Kafka to initialize
```

### 3. Test with Console Tools
```bash
# Copy client config into container
docker cp client-ssl.properties kafka:/tmp/client-ssl.properties

# Create a topic
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --topic secure-orders \
  --bootstrap-server localhost:9093 \
  --command-config /tmp/client-ssl.properties

# Produce a message
echo '{"test": "message"}' | docker exec -i kafka \
  /opt/kafka/bin/kafka-console-producer.sh \
  --topic secure-orders \
  --bootstrap-server localhost:9093 \
  --producer.config /tmp/client-ssl.properties

# Consume messages
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --topic secure-orders \
  --from-beginning \
  --bootstrap-server localhost:9093 \
  --consumer.config /tmp/client-ssl.properties
```

### 4. Test with Go Applications

Producer:
```bash
cd producer
go run main.go
```

Consumer (in a new terminal):
```bash
cd consumer
go run main.go
```

### 5. Cleanup
```bash
docker compose down
rm -rf secrets/
```

## What's Happening?

- **Port 9093**: SSL-encrypted Kafka broker
- **Port 8080**: Kafka UI (accessible via browser)
- Kafka enforces SSL for all client connections
- CA certificate (`secrets/ca-cert.pem`) validates the broker's identity
- No plain-text data transmission!

## Common Issues

**"Connection refused"**: Wait longer for Kafka to start (try 20-30 seconds)

**"Certificate verification failed"**: Ensure certificates were generated correctly and the CA cert path is correct

**Go import errors**: Run `go mod tidy` in the producer/consumer directory

## Next Steps

Read the full [README.md](./README.md) for:
- Detailed explanations of each step
- Mutual TLS (mTLS) configuration
- Security best practices
- Troubleshooting guide
