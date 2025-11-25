package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var (
	// ErrNotFound is returned when a record is not found.
	ErrNotFound = errors.New("record not found")

	// ErrInvalidID is returned when an invalid ID is provided.
	ErrInvalidID = errors.New("invalid ID")
)

// Bucket names.
const (
	BucketTokens     = "tokens"
	BucketDeals      = "deals"
	BucketJournals   = "journals"
	BucketWalletTxns = "wallet_txns"
	BucketReceipts   = "receipts"
)

// Store represents the bbolt database wrapper.
type Store struct {
	db *bolt.DB
}

// New creates a new Store instance and initializes buckets.
func New(dbPath string) (*Store, error) {
	db, err := bolt.Open(dbPath, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize buckets.
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := []string{BucketTokens, BucketDeals, BucketJournals, BucketWalletTxns, BucketReceipts}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// NextID generates the next ID for a bucket.
func (s *Store) NextID(bucketName string) (int64, error) {
	var id int64
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		seq, err := b.NextSequence()
		if err != nil {
			return err
		}
		id = int64(seq)
		return nil
	})
	return id, err
}

// Put stores a value in the specified bucket with the given key.
func (s *Store) Put(bucketName string, key int64, value interface{}) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}

		return b.Put(itob(key), data)
	})
}

// Get retrieves a value from the specified bucket with the given key.
func (s *Store) Get(bucketName string, key int64, value interface{}) error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		data := b.Get(itob(key))
		if data == nil {
			return ErrNotFound
		}

		return json.Unmarshal(data, value)
	})
}

// Delete removes a value from the specified bucket with the given key.
func (s *Store) Delete(bucketName string, key int64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		return b.Delete(itob(key))
	})
}

// List retrieves all values from the specified bucket.
func (s *Store) List(bucketName string, filter func(data []byte) bool) ([][]byte, error) {
	var results [][]byte

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		return b.ForEach(func(k, v []byte) error {
			if filter == nil || filter(v) {
				// Copy the value since it's only valid during the transaction.
				copied := make([]byte, len(v))
				copy(copied, v)
				results = append(results, copied)
			}
			return nil
		})
	})

	return results, err
}

// PutString stores a string value with a string key.
func (s *Store) PutString(bucketName, key, value string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		return b.Put([]byte(key), []byte(value))
	})
}

// GetString retrieves a string value with a string key.
func (s *Store) GetString(bucketName, key string) (string, error) {
	var value string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		data := b.Get([]byte(key))
		if data == nil {
			return ErrNotFound
		}

		value = string(data)
		return nil
	})
	return value, err
}

// DeleteString removes a value with a string key.
func (s *Store) DeleteString(bucketName, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketName)
		}

		return b.Delete([]byte(key))
	})
}

// itob converts an int64 to a byte slice for use as a bbolt key.
func itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
