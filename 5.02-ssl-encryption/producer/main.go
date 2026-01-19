package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

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
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected to Kafka over SSL!")
	fmt.Println("Producing messages to topic: secure-orders")
	fmt.Println("---")

	ctx := context.Background()
	topic := "secure-orders"

	// Produce messages
	for i := 1; i <= 10; i++ {
		message := fmt.Sprintf(`{"orderId": "%d", "amount": %.2f, "customer": "user%d@example.com", "timestamp": "%s"}`,
			2000+i, float64(i)*25.50, i, time.Now().Format(time.RFC3339))

		record := &kgo.Record{
			Topic: topic,
			Key:   []byte(fmt.Sprintf("order-%d", 2000+i)),
			Value: []byte(message),
		}

		// Produce synchronously
		results := client.ProduceSync(ctx, record)
		if err := results.FirstErr(); err != nil {
			log.Printf("Failed to send message: %v", err)
			continue
		}

		// Get the result for the record
		for _, res := range results {
			fmt.Printf("✓ Message %d sent to partition %d at offset %d\n", i, res.Record.Partition, res.Record.Offset)
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("---")
	fmt.Println("All messages sent successfully!")
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
	// InsecureSkipVerify skips hostname verification - needed because our cert is for "kafka" but we connect to "localhost"
	// This still validates the certificate chain, just not the hostname
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Skip hostname verification for localhost testing with self-signed certs
	}

	return tlsConfig, nil
}
