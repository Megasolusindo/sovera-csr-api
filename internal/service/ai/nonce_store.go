package ai

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NonceStore defines the interface for verifying single-use replay protection nonces.
type NonceStore interface {
	// VerifyAndMarkNonce checks if a nonce has been used. If not used, it marks it as used and returns true.
	// If already used, it returns false (replay detected).
	VerifyAndMarkNonce(nonce string, ttl time.Duration) (bool, error)
}

// InMemoryNonceStore provides a thread-safe, in-memory single-use nonce store with automatic TTL eviction.
type InMemoryNonceStore struct {
	mu     sync.Mutex
	nonces map[string]time.Time
}

func NewInMemoryNonceStore() *InMemoryNonceStore {
	store := &InMemoryNonceStore{
		nonces: make(map[string]time.Time),
	}
	// Start background janitor for TTL eviction
	go store.janitor(1 * time.Minute)
	return store
}

func (s *InMemoryNonceStore) VerifyAndMarkNonce(nonce string, ttl time.Duration) (bool, error) {
	if nonce == "" {
		return false, fmt.Errorf("nonce cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if exp, exists := s.nonces[nonce]; exists {
		if now.Before(exp) {
			return false, nil // Replay detected!
		}
	}

	s.nonces[nonce] = now.Add(ttl)
	return true, nil
}

func (s *InMemoryNonceStore) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now().UTC()
		for nonce, exp := range s.nonces {
			if now.After(exp) {
				delete(s.nonces, nonce)
			}
		}
		s.mu.Unlock()
	}
}

// RedisClientInterface decouples Redis client dependency for RedisNonceStore
type RedisClientInterface interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
}

// RedisNonceStore provides Redis-backed single-use nonce validation matching §9.2 Redis Nonce Optimization.
type RedisNonceStore struct {
	redisClient RedisClientInterface
	prefix      string
}

func NewRedisNonceStore(redisClient RedisClientInterface, prefix string) *RedisNonceStore {
	if prefix == "" {
		prefix = "nonce:task:"
	}
	return &RedisNonceStore{
		redisClient: redisClient,
		prefix:      prefix,
	}
}

func (r *RedisNonceStore) VerifyAndMarkNonce(nonce string, ttl time.Duration) (bool, error) {
	if nonce == "" {
		return false, fmt.Errorf("nonce cannot be empty")
	}
	if r.redisClient == nil {
		return false, fmt.Errorf("redis client is nil")
	}

	key := fmt.Sprintf("%s%s", r.prefix, nonce)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// SetNX (SET if Not eXists) atomically sets the key only if it doesn't already exist.
	// Returns true if key was set (valid first use), false if key already exists (replay).
	success, err := r.redisClient.SetNX(ctx, key, "1", ttl)
	if err != nil {
		return false, fmt.Errorf("redis setnx error: %w", err)
	}

	return success, nil
}
