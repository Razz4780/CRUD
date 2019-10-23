package storage

import (
	"sync"
)

type CmapStorage struct {
	values map[string]Data
	mut    sync.RWMutex
}

func (m CmapStorage) Put(key string, object []byte, contentType string) {
	m.mut.Lock()
	m.values[key] = Data{object, contentType}
	m.mut.Unlock()
}

func (m CmapStorage) Get(key string) (Data, error) {
	m.mut.RLock()
	val, exists := m.values[key]
	m.mut.RUnlock()
	var err error
	if !exists {
		err = KeyAbsentError
	}
	return val, err
}

func (m CmapStorage) Delete(key string) error {
	m.mut.Lock()
	_, existed := m.values[key]
	delete(m.values, key)
	m.mut.Unlock()
	if existed {
		return nil
	} else {
		return KeyAbsentError
	}
}

func (m CmapStorage) Keys() []string {
	m.mut.RLock()
	keys := make([]string, len(m.values))
	i := 0
	for key := range m.values {
		keys[i] = key
		i++
	}
	m.mut.RUnlock()
	return keys
}

func NewCmapStorage() CmapStorage {
	return CmapStorage{make(map[string]Data), sync.RWMutex{}}
}
