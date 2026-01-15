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
		kgo.ConsumerGroup("manual-commit-group"),
		kgo.ConsumeTopics("orders"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(), // NOTE: disable auto-commit for manual control
	)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer client.Close()

	log.Println("starting consumer...")

	ctx := context.Background()

	for {
		fetches := client.PollRecords(ctx, 5)
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("fetch errors: %v", errs)
		}

		log.Println("number of records fetched:", fetches.NumRecords())

		records := fetches.Records()

		for _, record := range records {
			log.Printf("processing message at offset %d", record.Offset)
			time.Sleep(500 * time.Millisecond)
			log.Printf("successfully processed message at offset %d", record.Offset)

			err = client.CommitRecords(ctx, record)
			if err != nil {
				return fmt.Errorf("commit error: %w", err)
			}
		}

	}
}
