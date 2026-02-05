package storage

import (
	"sync"
	"time"

	"github.com/openclaw/amp-relay-go/internal/protocol"
)

// MessageStore defines the interface for storing and retrieving AMP messages
type MessageStore interface {
	// Save stores a message with optional TTL
	Save(message *protocol.Message, ttl time.Duration) error

	// Get retrieves a message by ID
	Get(id string) (*protocol.Message, error)

	// Delete removes a message by ID
	Delete(id string) error

	// List returns all messages (with optional filtering in the future)
	List() ([]*protocol.Message, error)
}

// MemoryStore implements MessageStore in memory
type MemoryStore struct {
	messages map[string]*storedMessage
	mutex    sync.RWMutex
}

type storedMessage struct {
	message *protocol.Message
	expiry  time.Time
}

// NewMemoryStore creates a new in-memory message store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		messages: make(map[string]*storedMessage),
	}
}

// Save stores a message with optional TTL
func (ms *MemoryStore) Save(message *protocol.Message, ttl time.Duration) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	var expiry time.Time
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	} else {
		// No expiration if TTL is 0 or negative
		expiry = time.Time{}
	}

	ms.messages[message.ID] = &storedMessage{
		message: message,
		expiry:  expiry,
	}

	return nil
}

// Get retrieves a message by ID
func (ms *MemoryStore) Get(id string) (*protocol.Message, error) {
	ms.mutex.RLock()
	stored, exists := ms.messages[id]
	if !exists {
		ms.mutex.RUnlock()
		return nil, nil // Return nil if not found
	}

	// Check if message has expired
	if !stored.expiry.IsZero() && time.Now().After(stored.expiry) {
		ms.mutex.RUnlock()
		// Upgrade to write lock to remove expired message
		ms.mutex.Lock()
		// Double-check the message still exists and is still expired
		stored, exists = ms.messages[id]
		if exists && !stored.expiry.IsZero() && time.Now().After(stored.expiry) {
			delete(ms.messages, id)
		}
		ms.mutex.Unlock()
		return nil, nil
	}

	msg := stored.message
	ms.mutex.RUnlock()
	return msg, nil
}

// Delete removes a message by ID
func (ms *MemoryStore) Delete(id string) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	delete(ms.messages, id)
	return nil
}

// List returns all non-expired messages
func (ms *MemoryStore) List() ([]*protocol.Message, error) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	var result []*protocol.Message
	now := time.Now()

	for id, stored := range ms.messages {
		// Check if message has expired
		if !stored.expiry.IsZero() && now.After(stored.expiry) {
			// Remove expired message
			delete(ms.messages, id)
			continue
		}

		result = append(result, stored.message)
	}

	return result, nil
}
