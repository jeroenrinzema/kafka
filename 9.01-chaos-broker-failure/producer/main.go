package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudproud/graceful"
	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	ctx := graceful.NewContext(context.Background())
	fmt.Println("starting Resilient Producer...")
	fmt.Println("connected to: localhost:9092,localhost:9094,localhost:9095")

	// Configure resilient producer
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092", "localhost:9094", "localhost:9095"),
		kgo.DefaultProduceTopic("chaos-test"),
		// Producer resilience settings
		kgo.RequiredAcks(kgo.AllISRAcks()),         // Wait for all in-sync replicas
		kgo.ProducerBatchMaxBytes(16384),           // Batch size
		kgo.RequestTimeoutOverhead(10*time.Second), // Extra time for retries
		kgo.RetryBackoffFn(func(tries int) time.Duration {
			// Exponential backoff for retries
			return time.Duration(tries) * 100 * time.Millisecond
		}),
	)
	if err != nil {
		return err
	}

	defer client.Close()

	produced := 0
	errors := 0
	uptime := time.Now()

	ctx.Closer(func() {
		elapsed := time.Since(uptime)
		fmt.Printf("messages produced: %d\n", produced)
		fmt.Printf("errors: %d\n", errors)
		successRate := float64(produced-errors) / float64(produced) * 100
		fmt.Printf("success rate: %.2f%%\n", successRate)
		fmt.Printf("duration: %v\n", elapsed.Round(time.Second))
	})

	fmt.Println("producing messages every 100ms...")
	fmt.Println("watch for errors during broker failures!")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			produced++

			msg := fmt.Sprintf("message #%d at %s", produced, time.Now().Format("15:04:05.000"))
			key := fmt.Sprintf("key-%d", produced%10)

			record := &kgo.Record{
				Key:   []byte(key),
				Value: []byte(msg),
			}

			client.Produce(ctx, record, func(r *kgo.Record, err error) {
				if err != nil {
					errors++
					fmt.Printf("produce error: %v\n", err)
					return
				}

				if r.Offset%10 == 0 {
					fmt.Printf("produced message offset %d, partition %d\n", r.Offset, r.Partition)
				}
			})
		}
	}()

	ctx.AwaitKillSignal()
	return nil
}
