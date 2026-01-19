package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
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
	fmt.Println("Starting Split-Brain Test Consumer...")
	fmt.Println("Connected to: localhost:9092,localhost:9094,localhost:9095")
	fmt.Println()
	fmt.Println("This consumer helps detect split-brain symptoms.")
	fmt.Println("Watch for gaps in message sequences or partition changes!")
	fmt.Println()

	// Configure resilient consumer with settings for split-brain detection
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092", "localhost:9094", "localhost:9095"),
		kgo.ConsumeTopics("split-brain-test"),
		kgo.ConsumerGroup("split-brain-test-group"),
		// Longer timeouts to handle partition scenarios
		kgo.SessionTimeout(45*time.Second),
		kgo.HeartbeatInterval(5*time.Second),
		kgo.RebalanceTimeout(60*time.Second),
		// Manual commits for safety during chaos
		kgo.DisableAutoCommit(),
		// Fetch settings
		kgo.FetchMinBytes(1),
		kgo.FetchMaxWait(5*time.Second),
		// Quick metadata refresh to detect changes
		kgo.MetadataMinAge(1*time.Second),
	)
	if err != nil {
		return err
	}

	defer client.Close()

	var consumed int64
	var errors int64
	var lastOffset = make(map[int32]int64)
	var gaps int64
	uptime := time.Now()

	ctx.Closer(func() {
		elapsed := time.Since(uptime)
		fmt.Println()
		fmt.Println("=== Consumer Statistics ===")
		fmt.Printf("Messages consumed: %d\n", consumed)
		fmt.Printf("Errors encountered: %d\n", errors)
		fmt.Printf("Offset gaps detected: %d\n", gaps)
		fmt.Printf("Duration: %v\n", elapsed.Round(time.Second))

		if gaps > 0 {
			fmt.Println()
			fmt.Println("WARNING: Offset gaps may indicate message loss!")
			fmt.Println("This could be a symptom of split-brain with unclean leader election.")
		}
	})

	fmt.Println("Consuming messages...")
	fmt.Println("Legend: . = message, G = gap detected, R = rebalance, E = error")
	fmt.Println()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			fetches := client.PollFetches(ctx)

			// Check for errors (may indicate partition issues)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					atomic.AddInt64(&errors, 1)
					timestamp := time.Now().Format("15:04:05.000")
					fmt.Printf("\nE [%s] Poll error: topic=%s partition=%d: %v\n",
						timestamp, err.Topic, err.Partition, err.Err)
				}
				time.Sleep(1 * time.Second)
				continue
			}

			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				atomic.AddInt64(&consumed, 1)
				timestamp := time.Now().Format("15:04:05.000")

				// Check for offset gaps (potential split-brain symptom)
				if lastOff, exists := lastOffset[record.Partition]; exists {
					if record.Offset != lastOff+1 && record.Offset != 0 {
						atomic.AddInt64(&gaps, 1)
						fmt.Printf("\nG [%s] OFFSET GAP DETECTED on partition %d: expected %d, got %d (gap of %d messages)\n",
							timestamp, record.Partition, lastOff+1, record.Offset, record.Offset-lastOff-1)
					}
				}
				lastOffset[record.Partition] = record.Offset

				// Commit each record
				err := client.CommitRecords(ctx, record)
				if err != nil {
					atomic.AddInt64(&errors, 1)
					fmt.Printf("\nE [%s] Commit error for partition %d offset %d: %v\n",
						timestamp, record.Partition, record.Offset, err)
					continue
				}

				// Print progress every 10 messages
				if record.Offset%10 == 0 {
					fmt.Printf(". [%s] partition %d, offset %d, key=%s\n",
						timestamp, record.Partition, record.Offset, string(record.Key))
				}
			}
		}
	}()

	ctx.AwaitKillSignal()
	return nil
}
