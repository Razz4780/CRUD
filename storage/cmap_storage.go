package storage

import (
	"github.com/orcaman/concurrent-map"
)

type CmapStorage struct {
	values cmap.ConcurrentMap
}

func (m CmapStorage) Put(key string, object []byte, contentType string) {
	m.values.Set(key, Data{object, contentType})
}

func (m CmapStorage) Get(key string) (Data, error) {
	if val, exists := m.values.Get(key); exists {
		return val.(Data), nil
	} else {
		return Data{}, KeyAbsentError
	}
}

func (m CmapStorage) Delete(key string) error {
	if _, exists := m.values.Pop(key); exists {
		return nil
	} else {
		return KeyAbsentError
	}
}

func (m CmapStorage) Keys() []string {
	return m.values.Keys()
}

func NewCmapStorage() CmapStorage {
	return CmapStorage{cmap.New()}
}
