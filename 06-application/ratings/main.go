package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"time"

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
	client, err := kgo.NewClient(kgo.SeedBrokers("127.0.0.1:9092"))
	if err != nil {
		return err
	}

	ctx := graceful.NewContext(context.Background())
	go ctx.Closer(func() {
		client.Close()
	})

	go producer(ctx, client)

	ctx.AwaitKillSignal()
	return nil
}

type AppRating struct {
	Movie  string
	Rating int
}

func randRange(min, max int) int {
	return mrand.Intn(max-min) + min
}

var movies = []string{
	"Die Hard",
	"Spirited Away",
	"Mission Impossible",
}

func producer(ctx context.Context, client *kgo.Client) {
	for {
		time.Sleep(200 * time.Millisecond)
		key := make([]byte, 10)
		io.ReadFull(rand.Reader, key)

		n := mrand.Int() % len(movies)
		rating := AppRating{
			Movie:  movies[n],
			Rating: randRange(0, 100),
		}

		value, _ := json.Marshal(rating)

		record := &kgo.Record{
			Key:   key,
			Topic: "movie.ratings",
			Value: value,
		}

		client.Produce(ctx, record, nil)
		fmt.Println("Produced message to ratings topic at:", time.Now())
	}
}
