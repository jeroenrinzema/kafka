package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	maxRetries      = 5
	initialBackoff  = 1 * time.Second
	maxBackoff      = 30 * time.Second
	backoffMultiple = 2.0
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092"),
		kgo.ConsumerGroup("exponential-backoff-group"),
		kgo.ConsumeTopics("orders"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	log.Printf("starting consumer with exponential backoff (max retries: %d, initial: %v, multiplier: %.1fx)\n", maxRetries, initialBackoff, backoffMultiple)

	for {
		fetches := client.PollRecords(ctx, 10)
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("polling records: %v", errs)
		}

		log.Println("number of records fetched:", fetches.NumRecords())
		records := fetches.Records()

		for _, record := range records {
			log.Printf("received message: %q (partition=%d, offset=%d)\n", string(record.Value), record.Partition, record.Offset)

			err := processWithExponentialBackoff(record)
			if err != nil {
				log.Printf("failed to process message after %d retries: %v\n", maxRetries, err)
				log.Println("warning: message will be skipped")
			}

			err = client.CommitRecords(ctx, record)
			if err != nil {
				return fmt.Errorf("committing offset: %w", err)
			}

			log.Println("offset committed")
		}
	}
}

func processWithExponentialBackoff(record *kgo.Record) (err error) {
	for attempt := range maxRetries {
		log.Printf("processing attempt %d/%d\n", attempt, maxRetries)

		err = processMessage(record, attempt)
		if err == nil {
			log.Println("message processed successfully")
			return nil
		}

		log.Printf("attempt %d failed: %v\n", attempt, err)

		backoff := calculateBackoff(attempt)
		log.Printf("waiting %v before retry\n", backoff)
		time.Sleep(backoff)
	}

	return fmt.Errorf("all retry attempts exhausted: %w", err)
}

func calculateBackoff(attempt int) time.Duration {
	backoff := float64(initialBackoff) * math.Pow(backoffMultiple, float64(attempt-1))
	if backoff > float64(maxBackoff) {
		backoff = float64(maxBackoff)
	}
	return time.Duration(backoff)
}

func processMessage(record *kgo.Record, attempt int) error {
	time.Sleep(200 * time.Millisecond)

	message := string(record.Value)

	if strings.Contains(message, "transient") && attempt < 3 {
		return fmt.Errorf("simulated transient error (attempt %d, will succeed at attempt 3)", attempt)
	}

	if strings.Contains(message, "permanent") {
		return fmt.Errorf("simulated permanent error: invalid data format")
	}

	log.Printf("processing order: %s\n", message)
	return nil
}
