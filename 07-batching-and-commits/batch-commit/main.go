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
		kgo.ConsumerGroup("batch-commit-group"),
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
	consumed := 0

	for {
		fetches := client.PollRecords(ctx, 5)
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("fetch errors: %v", errs)
		}

		log.Println("number of records fetched:", fetches.NumRecords())

		records := fetches.Records()
		consumed += len(records)

		log.Printf("processing %d messages", len(records))
		time.Sleep(2 * time.Second)
		log.Printf("successfully processed message %d", consumed)

		err = client.CommitRecords(ctx, records...)
		if err != nil {
			return fmt.Errorf("commit error: %w", err)
		}
	}
}
