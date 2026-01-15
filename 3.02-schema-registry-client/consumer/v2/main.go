package main

import (
	"context"
	"encoding/binary"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hamba "github.com/hamba/avro/v2"
)

type UserV2 struct {
	ID        int32   `avro:"id"`
	Username  string  `avro:"username"`
	Email     string  `avro:"email"`
	CreatedAt int64   `avro:"created_at"`
	Phone     *string `avro:"phone"` // Optional field
}

func main() {
	// Configuration
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	schemaRegistryURL := getEnv("SCHEMA_REGISTRY_URL", "http://localhost:8081")
	topic := "users"
	groupID := "user-consumer-group-v2"

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": true,
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Subscribe to topic
	err = consumer.Subscribe(topic, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %v", err)
	}

	// Create Schema Registry client
	client, err := schemaregistry.NewClient(schemaregistry.NewConfig(schemaRegistryURL))
	if err != nil {
		log.Fatalf("Failed to create schema registry client: %v", err)
	}

	// Cache for schemas by ID
	schemaCache := make(map[int]*hamba.RecordSchema)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, closing consumer...")
		cancel()
	}()

	log.Printf("Starting consumer (v2), subscribed to topic: %s\n", topic)
	log.Println("Waiting for messages... (Press Ctrl+C to exit)")

	messageCount := 0

	// Consume messages
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down consumer...")
			return
		default:
			msg, err := consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Timeout is expected when no messages are available
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				log.Printf("Consumer error: %v\n", err)
				continue
			}

			// Parse Schema Registry wire format: [magic_byte] [schema_id] [avro_payload]
			if len(msg.Value) < 5 {
				log.Printf("Message too short to be valid Avro with Schema Registry format\n")
				continue
			}

			if msg.Value[0] != 0 {
				log.Printf("Invalid magic byte: expected 0, got %d\n", msg.Value[0])
				continue
			}

			schemaID := int(binary.BigEndian.Uint32(msg.Value[1:5]))
			avroPayload := msg.Value[5:]

			// Get schema from cache or Schema Registry
			avroSchema, ok := schemaCache[schemaID]
			if !ok {
				schemaInfo, err := client.GetBySubjectAndID(topic+"-value", schemaID)
				if err != nil {
					log.Printf("Failed to fetch schema %d: %v\n", schemaID, err)
					continue
				}

				parsedSchema, err := hamba.Parse(schemaInfo.Schema)
				if err != nil {
					log.Printf("Failed to parse schema: %v\n", err)
					continue
				}

				recordSchema, ok := parsedSchema.(*hamba.RecordSchema)
				if !ok {
					log.Printf("Schema is not a record schema\n")
					continue
				}

				schemaCache[schemaID] = recordSchema
				avroSchema = recordSchema
				log.Printf("Cached schema ID %d (version with %d fields)\n", schemaID, len(recordSchema.Fields()))
			}

			// Deserialize the Avro payload
			var user UserV2
			err = hamba.Unmarshal(avroSchema, avroPayload, &user)
			if err != nil {
				log.Printf("Failed to unmarshal Avro data: %v\n", err)
				continue
			}

			messageCount++
			createdTime := time.UnixMilli(user.CreatedAt)

			log.Printf("📨 Message %d | Partition: %d, Offset: %d | Schema ID: %d\n",
				messageCount,
				msg.TopicPartition.Partition,
				msg.TopicPartition.Offset,
				schemaID)
			log.Printf("   User ID: %d\n", user.ID)
			log.Printf("   Username: %s\n", user.Username)
			log.Printf("   Email: %s\n", user.Email)
			log.Printf("   Created: %s\n", createdTime.Format(time.RFC3339))
			if user.Phone != nil {
				log.Printf("   Phone: %s\n", *user.Phone)
			} else {
				log.Printf("   Phone: <not set>\n")
			}
			log.Println("   " + strings.Repeat("-", 50))
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
