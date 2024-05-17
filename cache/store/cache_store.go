package store

import (
	"time"
)

// CacheStore is the interface of a cache backend
type CacheStore interface {
	// Get retrieves an item from the cache. Returns the item or nil, and a bool indicating
	// whether the key was found.
	Get(key string, value interface{}) error

	// Set sets an item to the cache, replacing any existing item.
	Set(key string, value interface{}, expire time.Duration) error
}
