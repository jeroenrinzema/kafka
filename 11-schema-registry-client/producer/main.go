package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
)

// User represents our domain model
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

	// Create Kafka producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"client.id":         "user-producer",
		"acks":              "all",
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Create Schema Registry client
	client, err := schemaregistry.NewClient(schemaregistry.NewConfig(schemaRegistryURL))
	if err != nil {
		log.Fatalf("Failed to create schema registry client: %v", err)
	}

	// Create Avro serializer
	serializer, err := avro.NewGenericSerializer(client, serde.ValueSerde, avro.NewSerializerConfig())
	if err != nil {
		log.Fatalf("Failed to create serializer: %v", err)
	}

	// Read and register schema
	schemaBytes, err := os.ReadFile("../schemas/user.avsc")
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}

	// Sample users to produce
	users := []User{
		{ID: 1, Username: "alice", Email: "alice@example.com", CreatedAt: time.Now().Unix()},
		{ID: 2, Username: "bob", Email: "bob@example.com", CreatedAt: time.Now().Unix()},
		{ID: 3, Username: "charlie", Email: "charlie@example.com", CreatedAt: time.Now().Unix()},
		{ID: 4, Username: "diana", Email: "diana@example.com", CreatedAt: time.Now().Unix()},
		{ID: 5, Username: "eve", Email: "eve@example.com", CreatedAt: time.Now().Unix()},
	}

	// Delivery report handler
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Failed to deliver message: %v\n", ev.TopicPartition.Error)
				} else {
					log.Printf("Delivered message to %v [partition %d] at offset %v\n",
						*ev.TopicPartition.Topic,
						ev.TopicPartition.Partition,
						ev.TopicPartition.Offset)
				}
			}
		}
	}()

	// Produce messages
	for _, user := range users {
		// Convert user to Avro-compatible map
		userMap := map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"created_at": user.CreatedAt,
		}

		// Serialize the message
		payload, err := serializer.Serialize(topic, userMap)
		if err != nil {
			log.Printf("Failed to serialize user %s: %v\n", user.Username, err)
			continue
		}

		// Produce the message
		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          payload,
			Key:            []byte(fmt.Sprintf("%d", user.ID)),
		}, nil)

		if err != nil {
			log.Printf("Failed to produce message: %v\n", err)
			continue
		}

		log.Printf("Produced user: %s (%s)\n", user.Username, user.Email)
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for all messages to be delivered
	log.Println("Flushing remaining messages...")
	producer.Flush(15 * 1000)
	log.Println("All messages sent!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
