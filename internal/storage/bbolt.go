package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

// BBoltBackend implements StorageBackend using BBolt (embedded key-value store)
type BBoltBackend struct {
	db   *bolt.DB
	path string
}

// NewBBoltBackend creates a new BBolt storage backend
func NewBBoltBackend(path string) *BBoltBackend {
	if path == "" {
		path = "/data/gagos.db"
	}
	return &BBoltBackend{path: path}
}

func (b *BBoltBackend) Init() error {
	// Ensure directory exists
	dir := filepath.Dir(b.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	b.db, err = bolt.Open(b.path, 0600, nil)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Create all buckets
	err = b.db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range AllBuckets() {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create buckets: %w", err)
	}

	log.Info().Str("path", b.path).Str("type", "bbolt").Msg("Storage initialized")
	return nil
}

func (b *BBoltBackend) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

func (b *BBoltBackend) Type() string {
	return StorageTypeBBolt
}

func (b *BBoltBackend) Set(bucket, key string, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		return bkt.Put([]byte(key), value)
	})
}

func (b *BBoltBackend) Get(bucket, key string) ([]byte, error) {
	var data []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		v := bkt.Get([]byte(key))
		if v != nil {
			data = make([]byte, len(v))
			copy(data, v)
		}
		return nil
	})
	return data, err
}

func (b *BBoltBackend) Delete(bucket, key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		return bkt.Delete([]byte(key))
	})
}

func (b *BBoltBackend) List(bucket string) ([][]byte, error) {
	var items [][]byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		return bkt.ForEach(func(k, v []byte) error {
			data := make([]byte, len(v))
			copy(data, v)
			items = append(items, data)
			return nil
		})
	})
	return items, err
}

func (b *BBoltBackend) ListKeys(bucket string) ([]string, error) {
	var keys []string
	err := b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		if bkt == nil {
			return fmt.Errorf("bucket not found: %s", bucket)
		}
		return bkt.ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	return keys, err
}

// GetDB returns the underlying BBolt database (for backwards compatibility)
func (b *BBoltBackend) GetDB() *bolt.DB {
	return b.db
}
