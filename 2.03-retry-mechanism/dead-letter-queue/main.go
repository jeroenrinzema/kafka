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
		kgo.ConsumerGroup("dlq-group"),
		kgo.ConsumeTopics("orders"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	defer client.Close()

	log.Printf("starting consumer with dead letter queue (max retries: %d, dlq topic: orders-dlq)\n", maxRetries)

	ctx := context.Background()

	for {
		fetches := client.PollRecords(ctx, 10)
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("polling records: %v", errs)
		}

		records := fetches.Records()
		log.Println("number of records fetched:", fetches.NumRecords())

		for _, record := range records {
			log.Printf("processing message: %q (partition=%d, offset=%d)\n", string(record.Value), record.Partition, record.Offset)

			err := processWithRetry(ctx, client, record)
			if err != nil {
				log.Printf("failed to process message after %d retries: %v\n", maxRetries, err)
				continue
			}

			err = client.CommitRecords(ctx, record)
			if err != nil {
				return fmt.Errorf("committing offset: %w", err)
			}

			log.Println("offset committed")
		}
	}
}

func processWithRetry(ctx context.Context, client *kgo.Client, record *kgo.Record) (err error) {
	for attempt := range maxRetries {
		log.Printf("processing attempt %d/%d\n", attempt, maxRetries)

		err := processMessage(record)
		if err == nil {
			log.Println("message processed successfully")
			return nil
		}

		log.Printf("attempt %d failed: %v\n", attempt, err)
		log.Println("retrying immediately...")
	}

	return sendToDLQ(ctx, client, record)
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

func sendToDLQ(ctx context.Context, client *kgo.Client, record *kgo.Record) error {
	dlqRecord := &kgo.Record{
		Topic: "orders-dlq",
		Key:   record.Key,
		Value: record.Value,
		Headers: []kgo.RecordHeader{
			// NOTE: headers are flexible, add any metadata you find useful
			{Key: "original-topic", Value: []byte(record.Topic)},
			{Key: "original-partition", Value: []byte(fmt.Sprintf("%d", record.Partition))},
			{Key: "original-offset", Value: []byte(fmt.Sprintf("%d", record.Offset))},
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
		},
	}

	if err := client.ProduceSync(ctx, dlqRecord).FirstErr(); err != nil {
		return fmt.Errorf("producing to DLQ: %w", err)
	}

	return nil
}
