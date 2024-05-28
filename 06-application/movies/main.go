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
		kgo.ConsumerGroup("enricher"),
		kgo.ConsumeTopics("movie.ratings"),
	)
	if err != nil {
		return err
	}

	ctx := graceful.NewContext(context.Background())
	go ctx.Closer(func() {
		client.Close()
	})

	fmt.Println("consumer up and running!")
	go consumer(ctx, client)

	ctx.AwaitKillSignal()
	return nil
}

type AppRating struct {
	Movie  string
	Rating int
}

type EnrichedRating struct {
	Movie  string
	Year   int
	Rating int
}

var movies = map[string]int{
	"Die Hard":           2010,
	"Spirited Away":      2001,
	"Mission Impossible": 2010,
}

func consumer(ctx context.Context, client *kgo.Client) {
	for {
		fetch := client.PollFetches(ctx)
		if fetch.Err() != nil {
			fmt.Println("unexpected error while polling:", fetch.Err())
			return
		}

		iter := fetch.RecordIter()
		for !iter.Done() {
			record := iter.Next()

			rating := AppRating{}
			json.Unmarshal(record.Value, &rating)

			fmt.Printf("incoming rating: movie=%s rating=%d partition=%d \n", rating.Movie, rating.Rating, record.Partition)

			year := movies[rating.Movie]

			enriched := EnrichedRating{
				Movie:  rating.Movie,
				Rating: rating.Rating,
				Year:   year,
			}

			value, _ := json.Marshal(enriched)
			record = &kgo.Record{
				Key:   record.Key,
				Topic: "movie.enriched.ratings",
				Value: value,
			}

			client.Produce(ctx, record, nil)
		}
	}
}
