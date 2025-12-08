package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
		kgo.SeedBrokers("localhost:9092"),
		kgo.ConsumerGroup("order-processor"),
		kgo.ConsumeTopics("order-events"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer client.Close()

	fmt.Println("Starting to consume orders...")

	ctx := context.Background()
	processedOrders := 0

	for {
		fetches := client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			fmt.Printf("Error fetching: %v\n", errs)
			continue
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()

			var order OrderEvent
			if err := json.Unmarshal(record.Value, &order); err != nil {
				fmt.Printf("Error unmarshaling message: %v\n", err)
				continue
			}

			fmt.Printf("Processing order %s for user %s: $%.2f\n", order.OrderID, order.UserID, order.Amount)
			processedOrders++

			if processedOrders >= 10 {
				fmt.Println("Processed 10 orders, shutting down...")
				return nil
			}
		}
	}
}
