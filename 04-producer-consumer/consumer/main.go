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
		kgo.ConsumerGroup("report"),
		kgo.ConsumeTopics("purchases"),
		// kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)
	if err != nil {
		return err
	}

	defer client.Close()
	ctx := context.Background()

	fmt.Println("up and running!")

	for {
		fetch := client.PollFetches(ctx)
		if fetch.Err() != nil {
			return err
		}

		iter := fetch.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			fmt.Println("consuming record key:", string(record.Key), "value:", string(record.Value), "partition", record.Partition, "offset", record.Offset)
		}
	}
}
