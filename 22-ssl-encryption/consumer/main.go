package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	// Configure SSL/TLS
	tlsConfig, err := createTLSConfig("../secrets/ca-cert.pem")
	if err != nil {
		log.Fatalf("Failed to create TLS config: %v", err)
	}

	// Create client with SSL configuration
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9093"),
		kgo.DialTLSConfig(tlsConfig),
		kgo.ConsumeTopics("secure-orders"),
		kgo.ConsumerGroup("ssl-consumer-group"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected to Kafka over SSL!")
	fmt.Println("Consuming messages from topic: secure-orders")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("---")

	// Setup signal handler for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Message counter
	msgCount := 0

	// Consume messages in a goroutine
	go func() {
		for {
			fetches := client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					log.Printf("Error fetching: %v\n", err)
				}
			}

			fetches.EachPartition(func(p kgo.FetchTopicPartition) {
				for _, record := range p.Records {
					msgCount++
					fmt.Printf("✓ Message %d [Partition: %d, Offset: %d]\n", msgCount, record.Partition, record.Offset)
					fmt.Printf("  Key: %s\n", string(record.Key))
					fmt.Printf("  Value: %s\n", string(record.Value))
					fmt.Println("---")
				}
			})
		}
	}()

	// Wait for interrupt signal
	<-sigchan
	fmt.Println("\nShutting down consumer...")
	cancel()
	fmt.Printf("Total messages consumed: %d\n", msgCount)
}

// createTLSConfig creates a TLS configuration with the CA certificate
func createTLSConfig(caCertPath string) (*tls.Config, error) {
	// Load CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Create certificate pool and add CA cert
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false, // Set to true only for testing with self-signed certs and hostname mismatch
	}

	return tlsConfig, nil
}
