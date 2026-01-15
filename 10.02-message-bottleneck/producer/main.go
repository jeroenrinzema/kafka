package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Order struct {
	OrderID   string
	UserID    string
	Product   string
	Amount    float64
	Region    string
	Timestamp time.Time
}

func main() {
	// Create Kafka client
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092", "localhost:9093", "localhost:9094"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Number of messages to produce
	totalMessages := 1000
	regions := []string{"us-east", "us-west", "eu-west", "ap-south"}

	fmt.Printf("Producing %d messages to topic 'metrics'...\n", totalMessages)
	start := time.Now()

	var wg sync.WaitGroup
	regionCounts := make(map[string]int)
	var mu sync.Mutex

	// Produce messages concurrently
	for i := 0; i < totalMessages; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			region := regions[id%len(regions)]
			order := Order{
				OrderID:   fmt.Sprintf("order-%d", id),
				UserID:    fmt.Sprintf("user-%d", id%100),
				Product:   fmt.Sprintf("product-%d", id%50),
				Amount:    float64(10 + (id % 100)),
				Region:    region,
				Timestamp: time.Now(),
			}

			message := fmt.Sprintf(`{"order_id":"%s","user_id":"%s","product":"%s","amount":%.2f,"region":"%s","timestamp":"%s"}`,
				order.OrderID, order.UserID, order.Product, order.Amount, order.Region, order.Timestamp.Format(time.RFC3339))

			record := &kgo.Record{
				Topic: "metrics",
				Key:   []byte(region), // Partition by region
				Value: []byte(message),
			}

			ctx := context.Background()
			if err := client.ProduceSync(ctx, record).FirstErr(); err != nil {
				log.Printf("Failed to produce message: %v", err)
			}

			mu.Lock()
			regionCounts[region]++
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("\nCompleted in %s\n", elapsed)
	fmt.Printf("Messages per second: %.2f\n\n", float64(totalMessages)/elapsed.Seconds())
	fmt.Println("Messages per region:")
	for region, count := range regionCounts {
		fmt.Printf("  %s: %d messages\n", region, count)
	}
}
