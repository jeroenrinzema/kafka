package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
)

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
}

func main() {
	// Configuration
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	schemaRegistryURL := getEnv("SCHEMA_REGISTRY_URL", "http://localhost:8081")
	topic := "users"
	groupID := "user-consumer-group"

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

	// Create Avro deserializer
	deserializer, err := avro.NewGenericDeserializer(client, serde.ValueSerde, avro.NewDeserializerConfig())
	if err != nil {
		log.Fatalf("Failed to create deserializer: %v", err)
	}

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

	log.Printf("Starting consumer, subscribed to topic: %s\n", topic)
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

			// Deserialize the message
			value, err := deserializer.Deserialize(topic, msg.Value)
			if err != nil {
				log.Printf("Failed to deserialize message: %v\n", err)
				continue
			}

			// Cast to map and extract fields
			userMap, ok := value.(map[string]interface{})
			if !ok {
				log.Printf("Unexpected value type: %T\n", value)
				continue
			}

			// Extract user data
			user := User{
				ID:        int(userMap["id"].(int32)),
				Username:  userMap["username"].(string),
				Email:     userMap["email"].(string),
				CreatedAt: userMap["created_at"].(int64),
			}

			messageCount++
			createdTime := time.Unix(user.CreatedAt, 0)

			log.Printf("📨 Message %d | Partition: %d, Offset: %d\n",
				messageCount,
				msg.TopicPartition.Partition,
				msg.TopicPartition.Offset)
			log.Printf("   User ID: %d\n", user.ID)
			log.Printf("   Username: %s\n", user.Username)
			log.Printf("   Email: %s\n", user.Email)
			log.Printf("   Created: %s\n", createdTime.Format(time.RFC3339))
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
