package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jeroenrinzema/application/pkg/graceful"
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
		kgo.ConsumerGroup("backup"),
		kgo.ConsumeTopics("movie.ratings"),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return err
	}

	ctx := graceful.NewContext(context.Background())
	go ctx.Closer(func() {
		client.Close()
	})

	file, err := os.OpenFile("export-topic", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	go ctx.Closer(func() {
		file.Close()
	})

	fmt.Println("consumer up and running!")
	go consumer(ctx, client, file)

	fmt.Println("up and running!")

	ctx.AwaitKillSignal()
	return nil
}

func consumer(ctx context.Context, client *kgo.Client, file *os.File) {
	for {
		fetch := client.PollFetches(ctx)
		if fetch.Err() != nil {
			fmt.Println("unexpected error while polling", fetch.Err())
			continue
		}

		records := fetch.Records()

		fmt.Println("comsuming records", len(records))

		backup, _ := json.Marshal(records)
		if _, err := file.Write(backup); err != nil {
			panic(err)
		}

		client.CommitRecords(ctx, records...)
	}
}
