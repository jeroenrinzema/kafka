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
	fmt.Println("starting Resilient Consumer...")
	fmt.Println("connected to: localhost:9092,localhost:9094,localhost:9095")

	// Configure resilient consumer
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092", "localhost:9094", "localhost:9095"),
		kgo.ConsumeTopics("chaos-test"),
		kgo.ConsumerGroup("chaos-test-group"),
		kgo.SessionTimeout(30*time.Second),   // Longer session timeout
		kgo.HeartbeatInterval(3*time.Second), // Regular heartbeats
		kgo.RebalanceTimeout(60*time.Second), // Time for rebalance
		kgo.DisableAutoCommit(),              // Manual commits for safety
	)
	if err != nil {
		return err
	}

	defer client.Close()

	consumed := 0
	errors := 0
	uptime := time.Now()

	ctx.Closer(func() {
		elapsed := time.Since(uptime)
		fmt.Printf("messages consumed: %d\n", consumed)
		fmt.Printf("errors: %d\n", errors)
		fmt.Printf("duration: %v\n", elapsed.Round(time.Second))
	})

	fmt.Println("consuming messages...")
	fmt.Println("watch for pauses during broker failures!")

	go func() {
	poll:
		for {
			fetches := client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					errors++
					fmt.Printf("poll error: %v\n", err)
				}
				time.Sleep(1 * time.Second)
				continue
			}

			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				consumed++
				err = client.CommitRecords(ctx, record)
				if err != nil {
					errors++
					fmt.Printf("commit error: %v\n", err)
					continue poll
				}

				if record.Offset%10 == 0 {
					fmt.Printf("consumed message offset %d, partition %d\n", record.Offset, record.Partition)
				}
			}
		}
	}()

	ctx.AwaitKillSignal()
	return nil
}
