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
	client, err := kgo.NewClient(kgo.SeedBrokers("127.0.0.1:9092"))
	if err != nil {
		return err
	}

	defer client.Close()

	ctx := context.Background()
	record := &kgo.Record{
		Key:   []byte("cart-abc"),
		Topic: "purchases",
		Value: []byte("Milk"),
	}

	result := client.ProduceSync(ctx, record)
	if result.FirstErr() != nil {
		return result.FirstErr()
	}

	fmt.Println("Produced message to purchases topic")
	return nil
}
