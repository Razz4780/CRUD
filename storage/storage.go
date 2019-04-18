package storage

import "errors"

// Data stored in storage.
type Data struct {
	Object      []byte
	ContentType string
}

type Storage interface {
	// Put places data in storage under given key.
	Put(key string, object []byte, contentType string)

	// Get retrieves from storage data under given key.
	// Returns KeyAbsentError if the key is not present.
	Get(key string) (Data, error)

	// Delete removes from storage data under given key.
	// Returns KeyAbsentError if the key is not present.
	Delete(key string) error

	// Keys lists keys present in storage.
	Keys() []string
}

var KeyAbsentError = errors.New("key not in storage")

func NewStorage() CmapStorage {
	return NewCmapStorage()
}
