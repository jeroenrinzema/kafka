package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const maxRetries = 3

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092"),
		kgo.ConsumerGroup("simple-retry-group"),
		kgo.ConsumeTopics("orders"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	log.Printf("starting consumer with simple retry mechanism (max retries: %d)\n", maxRetries)

	for {
		fetches := client.PollRecords(ctx, 10)
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("polling records: %v", errs)
		}

		log.Println("number of records fetched:", fetches.NumRecords())
		records := fetches.Records()

		for _, record := range records {
			log.Printf("received message: %q (partition=%d, offset=%d)\n", string(record.Value), record.Partition, record.Offset)

			err = processWithRetry(record)
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

func processWithRetry(record *kgo.Record) (err error) {
	for attempt := range maxRetries {
		log.Printf("processing attempt %d/%d\n", attempt, maxRetries)

		err = processMessage(record)
		if err == nil {
			log.Println("message processed successfully")
			return nil
		}

		log.Printf("attempt %d failed: %v\n", attempt, err)
	}

	return fmt.Errorf("all retry attempts exhausted: %w", err)
}

func processMessage(record *kgo.Record) error {
	time.Sleep(200 * time.Millisecond)

	message := string(record.Value)
	if strings.Contains(message, "fail") {
		return fmt.Errorf("simulated error: message contains 'fail'")
	}

	log.Printf("processing order: %s\n", message)
	return nil
}
