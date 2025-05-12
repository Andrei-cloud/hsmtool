package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// ZMK is Zone Master Key.
	ZMK KeyType = "ZMK"
	// ZPK is Zone PIN Key.
	ZPK KeyType = "ZPK"
	// TMK is Terminal Master Key.
	TMK KeyType = "TMK"
	// PVK is PIN Verification Key.
	PVK KeyType = "PVK"
	// KEK is Key Encryption Key.
	KEK KeyType = "KEK"
)

// KeyType represents the type of cryptographic key.
type KeyType string

// KeyEntry represents a stored key record.
type KeyEntry struct {
	Name       string    `json:"name"`
	Type       KeyType   `json:"type"`
	Length     int       `json:"length"`
	CheckValue string    `json:"check_value"`
	CreatedAt  time.Time `json:"created_at"`
}

// KeyStore manages key storage.
type KeyStore struct {
	mu       sync.RWMutex
	keys     map[string]KeyEntry
	filePath string
}

// NewKeyStore creates a new key store instance.
func NewKeyStore(storePath string) (*KeyStore, error) {
	if err := os.MkdirAll(filepath.Dir(storePath), 0o750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	ks := &KeyStore{
		keys:     make(map[string]KeyEntry),
		filePath: storePath,
	}

	// Load existing keys if any.
	if err := ks.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load keys: %v", err)
	}

	return ks, nil
}

// Store adds or updates a key entry.
func (ks *KeyStore) Store(entry KeyEntry) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if entry.Name == "" {
		return errors.New("key name cannot be empty")
	}

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	ks.keys[entry.Name] = entry

	return ks.save()
}

// Get retrieves a key entry by name.
func (ks *KeyStore) Get(name string) (KeyEntry, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	entry, exists := ks.keys[name]

	return entry, exists
}

// List returns all stored key entries.
func (ks *KeyStore) List() []KeyEntry {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	entries := make([]KeyEntry, 0, len(ks.keys))
	for _, entry := range ks.keys {
		entries = append(entries, entry)
	}

	return entries
}

// Delete removes a key entry.
func (ks *KeyStore) Delete(name string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if _, exists := ks.keys[name]; !exists {
		return errors.New("key not found")
	}

	delete(ks.keys, name)

	return ks.save()
}

// load reads key entries from storage file.
func (ks *KeyStore) load() error {
	data, err := os.ReadFile(ks.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &ks.keys)
}

// save writes key entries to storage file.
func (ks *KeyStore) save() error {
	data, err := json.MarshalIndent(ks.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %v", err)
	}

	return os.WriteFile(ks.filePath, data, 0o600)
}
