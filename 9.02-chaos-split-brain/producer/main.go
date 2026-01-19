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
	fmt.Println("Starting Split-Brain Test Producer...")
	fmt.Println("Connected to: localhost:9092,localhost:9094,localhost:9095")
	fmt.Println()
	fmt.Println("This producer helps demonstrate split-brain scenarios.")
	fmt.Println("Watch for errors during network partitions!")
	fmt.Println()

	// Configure resilient producer with settings that help detect split-brain
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092", "localhost:9094", "localhost:9095"),
		kgo.DefaultProduceTopic("split-brain-test"),
		// CRITICAL: Wait for all in-sync replicas to prevent data loss
		kgo.RequiredAcks(kgo.AllISRAcks()),
		// Batch settings
		kgo.ProducerBatchMaxBytes(16384),
		kgo.ProducerLinger(10*time.Millisecond),
		// Timeout settings - allow time for leader election
		kgo.RequestTimeoutOverhead(15*time.Second),
		kgo.ProduceRequestTimeout(30*time.Second),
		// Retry settings with exponential backoff
		kgo.RetryBackoffFn(func(tries int) time.Duration {
			backoff := time.Duration(tries) * 200 * time.Millisecond
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
			return backoff
		}),
		// Metadata refresh to detect partition changes quickly
		kgo.MetadataMinAge(1*time.Second),
	)
	if err != nil {
		return err
	}

	defer client.Close()

	var produced int64
	var succeeded int64
	var failed int64
	var lastPartition int32 = -1
	uptime := time.Now()

	ctx.Closer(func() {
		elapsed := time.Since(uptime)
		fmt.Println()
		fmt.Println("=== Producer Statistics ===")
		fmt.Printf("Messages attempted: %d\n", produced)
		fmt.Printf("Messages succeeded: %d\n", succeeded)
		fmt.Printf("Messages failed: %d\n", failed)
		if succeeded > 0 {
			successRate := float64(succeeded) / float64(produced) * 100
			fmt.Printf("Success rate: %.2f%%\n", successRate)
		}
		fmt.Printf("Duration: %v\n", elapsed.Round(time.Second))
	})

	fmt.Println("Producing messages every 200ms...")
	fmt.Println("Legend: + = success, X = failure, P = partition change")
	fmt.Println()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			msgNum := atomic.AddInt64(&produced, 1)
			timestamp := time.Now().Format("15:04:05.000")

			msg := fmt.Sprintf("split-brain-msg-%d at %s", msgNum, timestamp)
			key := fmt.Sprintf("key-%d", msgNum%10)

			record := &kgo.Record{
				Key:   []byte(key),
				Value: []byte(msg),
			}

			client.Produce(ctx, record, func(r *kgo.Record, err error) {
				if err != nil {
					atomic.AddInt64(&failed, 1)
					fmt.Printf("\nX [%s] FAILED msg #%d: %v\n", timestamp, msgNum, err)
					return
				}

				atomic.AddInt64(&succeeded, 1)

				// Detect partition changes (may indicate leader election)
				if lastPartition != -1 && lastPartition != r.Partition {
					fmt.Printf("\nP [%s] Partition change detected: %d -> %d (possible leader election)\n",
						timestamp, lastPartition, r.Partition)
				}
				lastPartition = r.Partition

				// Print progress every 10 messages
				if r.Offset%10 == 0 {
					fmt.Printf("+ [%s] msg #%d -> partition %d, offset %d\n",
						timestamp, msgNum, r.Partition, r.Offset)
				}
			})
		}
	}()

	ctx.AwaitKillSignal()
	return nil
}
