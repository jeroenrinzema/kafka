package main

import (
	"context"
	"fmt"
	"os"

	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	client, err := kgo.NewClient(
		kgo.SeedBrokers("127.0.0.1:9092"),
		kgo.ConsumerGroup("cart"),
		kgo.ConsumeTopics("purchases"),
	)
	if err != nil {
		return err
	}

	defer client.Close()
	ctx := context.Background()

	for {
		fetch := client.PollFetches(ctx)
		if fetch.Err() != nil {
			return err
		}

		iter := fetch.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			fmt.Println("consuming record key:", string(record.Key), "value:", string(record.Value))
		}
	}
}
