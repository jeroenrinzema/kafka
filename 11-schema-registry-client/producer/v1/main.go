package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	hamba "github.com/hamba/avro/v2"
)

// User represents our domain model
type User struct {
	ID        int32  `avro:"id"`
	Username  string `avro:"username"`
	Email     string `avro:"email"`
	CreatedAt int64  `avro:"created_at"`
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

	// Read schema
	schemaBytes, err := os.ReadFile("../schemas/user.avsc")
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	// Parse schema using hamba/avro
	avroSchema, err := hamba.Parse(string(schemaBytes))
	if err != nil {
		log.Fatalf("Failed to parse schema: %v", err)
	}

	// Register schema with Schema Registry
	schemaInfo := schemaregistry.SchemaInfo{
		Schema:     string(schemaBytes),
		SchemaType: "AVRO",
	}

	schemaID, err := client.Register(topic+"-value", schemaInfo, false)
	if err != nil {
		// Try to get existing schema
		schema, err2 := client.GetLatestSchemaMetadata(topic + "-value")
		if err2 != nil {
			log.Fatalf("Failed to register or get schema: register error: %v, get error: %v", err, err2)
		}
		schemaID = schema.ID
		log.Printf("Using existing schema ID: %d\n", schemaID)
	} else {
		log.Printf("Registered new schema with ID: %d\n", schemaID)
	}

	// Sample users to produce
	users := []User{
		{ID: 1, Username: "alice", Email: "alice@example.com", CreatedAt: time.Now().UnixMilli()},
		{ID: 2, Username: "bob", Email: "bob@example.com", CreatedAt: time.Now().UnixMilli()},
		{ID: 3, Username: "charlie", Email: "charlie@example.com", CreatedAt: time.Now().UnixMilli()},
		{ID: 4, Username: "diana", Email: "diana@example.com", CreatedAt: time.Now().UnixMilli()},
		{ID: 5, Username: "eve", Email: "eve@example.com", CreatedAt: time.Now().UnixMilli()},
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
		// Serialize using hamba/avro
		avroBytes, err := hamba.Marshal(avroSchema, user)
		if err != nil {
			log.Printf("Failed to marshal user %s: %v\n", user.Username, err)
			continue
		}

		// Create Schema Registry wire format: [magic_byte (0x00)] [schema_id (4 bytes)] [avro_payload]
		payload := make([]byte, 5+len(avroBytes))
		payload[0] = 0 // Magic byte
		binary.BigEndian.PutUint32(payload[1:5], uint32(schemaID))
		copy(payload[5:], avroBytes)

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
