package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type OrderEvent struct {
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Timestamp string  `json:"timestamp"`
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9093"),
		kgo.RequiredAcks(kgo.NoAck()),
	)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer client.Close()

	orders := []OrderEvent{
		{OrderID: "order-1", UserID: "user-1", Amount: 99.99, Status: "pending", Timestamp: time.Now().Format(time.RFC3339)},
		{OrderID: "order-2", UserID: "user-2", Amount: 149.50, Status: "pending", Timestamp: time.Now().Format(time.RFC3339)},
		{OrderID: "order-3", UserID: "user-1", Amount: 79.99, Status: "pending", Timestamp: time.Now().Format(time.RFC3339)},
		{OrderID: "order-4", UserID: "user-3", Amount: 299.99, Status: "pending", Timestamp: time.Now().Format(time.RFC3339)},
		{OrderID: "order-5", UserID: "user-2", Amount: 49.99, Status: "pending", Timestamp: time.Now().Format(time.RFC3339)},
	}

	ctx := context.Background()
	fmt.Println("Starting to publish messages...")

	for _, order := range orders {
		value, err := json.Marshal(order)
		if err != nil {
			fmt.Printf("Failed to marshal order: %v\n", err)
			continue
		}

		record := &kgo.Record{
			Topic: "orders",
			Key:   []byte(order.OrderID),
			Value: value,
		}

		results := client.ProduceSync(ctx, record)
		if err := results.FirstErr(); err != nil {
			fmt.Printf("Failed to produce order: %v\n", err)
			return err
		}

		fmt.Printf("Produced order: %s\n", order.OrderID)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("All orders produced successfully!")
	return nil
}
