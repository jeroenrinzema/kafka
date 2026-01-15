package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// UserProfile represents a user profile
type UserProfile struct {
	UserID  string `json:"userId"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Age     int    `json:"age"`
	Version int    `json:"version"`
}

// StateStore holds the current state
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

func (s *StateStore) Set(userID string, profile *UserProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.profiles[userID] = profile
}

func (s *StateStore) Delete(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.profiles, userID)
}

func (s *StateStore) Get(userID string) (*UserProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, exists := s.profiles[userID]
	return profile, exists
}

func (s *StateStore) GetAll() []*UserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profiles := make([]*UserProfile, 0, len(s.profiles))
	for _, v := range s.profiles {
		profiles = append(profiles, v)
	}
	return profiles
}

func (s *StateStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.profiles)
}

// QueryService provides HTTP API for the state store
type QueryService struct {
	store *StateStore
}

func (qs *QueryService) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	profiles := qs.store.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

func (qs *QueryService) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/users/"):]
	w.Header().Set("Content-Type", "application/json")
	profile, exists := qs.store.Get(userID)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}
	json.NewEncoder(w).Encode(profile)
}

func (qs *QueryService) handleGetCount(w http.ResponseWriter, r *http.Request) {
	count := qs.store.Count()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

func (qs *QueryService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	fmt.Printf("🚀 Starting Query Service on port %s...\n", port)

	store := NewStateStore()

	// Build state in background
	go buildStateFromKafka(store)

	// Setup HTTP handlers
	qs := &QueryService{store: store}
	http.HandleFunc("/users", qs.handleGetUsers)
	http.HandleFunc("/users/", qs.handleGetUser)
	http.HandleFunc("/users/count", qs.handleGetCount)
	http.HandleFunc("/health", qs.handleHealth)

	// Wait a bit for initial state load
	time.Sleep(2 * time.Second)

	fmt.Println()
	fmt.Println("📡 API Endpoints:")
	fmt.Printf("  GET  http://localhost:%s/users        - List all users\n", port)
	fmt.Printf("  GET  http://localhost:%s/users/:id    - Get specific user\n", port)
	fmt.Printf("  GET  http://localhost:%s/users/count  - Get user count\n", port)
	fmt.Printf("  GET  http://localhost:%s/health       - Health check\n", port)
	fmt.Println()

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func buildStateFromKafka(store *StateStore) {
	fmt.Println("📖 Building state from Kafka topic 'user-profiles'...")

	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092"),
		kgo.ConsumeTopics("user-profiles"),
		kgo.ConsumerGroup("query-service-"+time.Now().Format("20060102150405")),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		log.Printf("❌ Failed to create Kafka client: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	messagesProcessed := 0
	initialLoadComplete := false

	for {
		fetches := client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				log.Printf("❌ Fetch error: %v", err)
			}
			continue
		}

		iter := fetches.RecordIter()
		recordsInBatch := 0
		for !iter.Done() {
			record := iter.Next()
			recordsInBatch++
			messagesProcessed++

			userID := string(record.Key)

			if record.Value == nil {
				store.Delete(userID)
				continue
			}

			var profile UserProfile
			if err := json.Unmarshal(record.Value, &profile); err != nil {
				log.Printf("⚠️  Failed to parse profile: %v", err)
				continue
			}

			store.Set(userID, &profile)
		}

		if !initialLoadComplete && recordsInBatch > 0 {
			initialLoadComplete = true
			fmt.Printf("✅ Initial state loaded: %d messages processed, %d users in store\n",
				messagesProcessed, store.Count())
		}

		if recordsInBatch == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
