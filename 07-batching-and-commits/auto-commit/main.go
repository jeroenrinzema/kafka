package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092"),
		kgo.ConsumerGroup("auto-commit-group"),
		kgo.ConsumeTopics("orders"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.AutoCommitInterval(3*time.Second), // NOTE: auto-commit every 3 seconds
	)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer client.Close()

	log.Println("starting consumer...")

	ctx := context.Background()
	consumed := 0

	for {
		fetches := client.PollRecords(ctx, 5)
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("fetch errors: %v", errs)
		}

		log.Println("number of records fetched:", fetches.NumRecords())

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			consumed++
			log.Printf("processing message %d: %s", consumed, string(record.Value))

			if consumed == 8 {
				log.Printf("note: auto-commit may have already committed offset %d", record.Offset)
				return fmt.Errorf("simulated crash on message %d", consumed)
			}

			// Simulate slow processing (2 seconds per message)
			time.Sleep(2 * time.Second)
			log.Printf("successfully processed message %d", consumed)
		}
	}
}
