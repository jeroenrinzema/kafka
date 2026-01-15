package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// UserProfile represents a user profile in the state store
type UserProfile struct {
	UserID  string `json:"userId"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Age     int    `json:"age"`
	Version int    `json:"version"`
}

// StateStore holds the current state built from the compacted topic
type StateStore struct {
	mu       sync.RWMutex
	profiles map[string]*UserProfile
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		profiles: make(map[string]*UserProfile),
	}
}

// Set adds or updates a profile in the state store
func (s *StateStore) Set(userID string, profile *UserProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles[userID] = profile
	fmt.Printf("✓ Updated state: %s -> %s (v%d)\n", userID, profile.Name, profile.Version)
}

// Delete removes a profile from the state store (tombstone handling)
func (s *StateStore) Delete(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.profiles, userID)
	fmt.Printf("✗ Deleted from state: %s\n", userID)
}

// Get retrieves a profile from the state store
func (s *StateStore) Get(userID string) (*UserProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, exists := s.profiles[userID]
	return profile, exists
}

// GetAll returns all profiles in the state store
func (s *StateStore) GetAll() map[string]*UserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to avoid race conditions
	copy := make(map[string]*UserProfile, len(s.profiles))
	for k, v := range s.profiles {
		copy[k] = v
	}
	return copy
}

// Count returns the number of profiles in the state store
func (s *StateStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.profiles)
}

func main() {
	fmt.Println("🚀 Starting State Store Consumer...")
	fmt.Println("Building local cache from compacted topic 'user-profiles'")
	fmt.Println()

	// Create state store
	store := NewStateStore()

	// Create Kafka consumer
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092"),
		kgo.ConsumeTopics("user-profiles"),
		kgo.ConsumerGroup("state-store-consumer"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()), // Always start from beginning
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n🛑 Shutting down gracefully...")
		cancel()
	}()

	// Stats
	startTime := time.Now()
	messagesProcessed := 0
	initialLoadComplete := false

	fmt.Println("📖 Reading all messages from beginning to build state...")
	fmt.Println()

	// Consume messages
	for {
		select {
		case <-ctx.Done():
			printStateSummary(store, messagesProcessed, time.Since(startTime))
			return
		default:
			fetches := client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					fmt.Printf("❌ Fetch error: %v\n", err)
				}
				continue
			}

			// Process each record
			iter := fetches.RecordIter()
			recordsInBatch := 0
			for !iter.Done() {
				record := iter.Next()
				recordsInBatch++
				messagesProcessed++

				// Extract key (user ID)
				userID := string(record.Key)

				// Check for tombstone (null value = deletion)
				if record.Value == nil {
					store.Delete(userID)
					continue
				}

				// Parse profile
				var profile UserProfile
				if err := json.Unmarshal(record.Value, &profile); err != nil {
					fmt.Printf("⚠️  Failed to parse profile for %s: %v\n", userID, err)
					continue
				}

				// Update state store
				store.Set(userID, &profile)
			}

			// After first batch, consider initial load complete
			if !initialLoadComplete && recordsInBatch > 0 {
				initialLoadComplete = true
				fmt.Println()
				fmt.Println("✅ Initial state load complete!")
				fmt.Println()
				printStateSummary(store, messagesProcessed, time.Since(startTime))
				fmt.Println()
				fmt.Println("👀 Watching for updates... (Press Ctrl+C to exit)")
				fmt.Println()
			}

			// Small delay to avoid tight loop when no messages
			if recordsInBatch == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func printStateSummary(store *StateStore, messagesProcessed int, duration time.Duration) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 State Store Summary\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Messages processed: %d\n", messagesProcessed)
	fmt.Printf("Current state size: %d users\n", store.Count())
	fmt.Printf("Processing time: %v\n", duration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("Current State (Latest values only):")
	fmt.Println("─────────────────────────────────────────────")
	profiles := store.GetAll()
	if len(profiles) == 0 {
		fmt.Println("  (empty)")
	} else {
		for userID, profile := range profiles {
			fmt.Printf("  • %s: %s <%s> (age: %d, version: %d)\n",
				userID, profile.Name, profile.Email, profile.Age, profile.Version)
		}
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
