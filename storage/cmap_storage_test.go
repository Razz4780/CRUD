package storage

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestCmapStorage_Put(t *testing.T) {
	dataSetsSimple := []struct {
		key         string
		object      []byte
		contentType string
	}{
		{"key", []byte{}, ""},
		{"0", []byte{1, 2, 1, 2}, "type"},
		{"asdf1234", []byte{123, 200, 6}, "nums"},
	}

	for i, dataSet := range dataSetsSimple {
		t.Run(fmt.Sprint("simple set ", i), func(t *testing.T) {
			dataStorage := NewCmapStorage()
			dataStorage.Put(dataSet.key, dataSet.object, dataSet.contentType)

			val, exists := dataStorage.values.Get(dataSet.key)
			if !exists {
				t.Fatalf("key not present in storage")
			}
			data, ok := val.(Data)
			if !ok {
				t.Fatalf("stored value not of type Data")
			}
			if !exists {
				t.Fatalf("key not present in storage")
			}
			if !bytes.Equal(data.Object, dataSet.object) {
				t.Errorf("stored object differs: %v", data.Object)
			}
			if dataSet.contentType != data.ContentType {
				t.Fatalf("stored content type differs: %v", data.ContentType)
			}
			if len(dataStorage.values.Keys()) != 1 {
				t.Fatalf("storage size not equal to 1 after 1 put operation")
			}
		})
	}

	t.Run("multiple puts", func(t *testing.T) {
		dataStorage := NewCmapStorage()

		key1 := "key1"
		key2 := "key2"
		dataStorage.Put(key1, []byte{}, "")
		dataStorage.Put(key2, []byte{}, "")

		if len(dataStorage.values.Keys()) != 2 {
			t.Fatalf("invalid storage size")
		}
		if !dataStorage.values.Has(key1) {
			t.Fatalf("first key not present")
		}
		if !dataStorage.values.Has(key2) {
			t.Fatalf("second key not present")
		}
	})

	t.Run("same key puts", func(t *testing.T) {
		dataStorage := NewCmapStorage()

		key := "key"
		object := []byte{1, 2, 3, 4}
		contentType := "type"
		dataStorage.Put(key, []byte{}, "")
		dataStorage.Put(key, object, contentType)

		if len(dataStorage.values.Keys()) != 1 {
			t.Fatalf("invalid storage size")
		}
		val, exists := dataStorage.values.Get(key)
		if !exists {
			t.Fatal("key not present")
		} else {
			data := val.(Data)
			if data.ContentType != contentType {
				t.Errorf("stored content type differs: %v", data.ContentType)
			}
			if !bytes.Equal(data.Object, object) {
				t.Errorf("stored object differs: %v", data.Object)
			}
		}
	})
}

func TestCmapStorage_Get(t *testing.T) {
	dataStorage := NewCmapStorage()

	key := "key"
	data := Data{[]byte{}, ""}

	dataStorage.values.Set(key, data)

	extractedData, err := dataStorage.Get(key)
	if err != nil {
		t.Fatalf("get operation failed")
	}
	if !bytes.Equal(extractedData.Object, data.Object) {
		t.Fatalf("extracted object differs: %v", extractedData.Object)
	}
	if extractedData.ContentType != data.ContentType {
		t.Fatalf("extracted content type differs: %v", extractedData.ContentType)
	}
}

func TestCmapStorage_Delete(t *testing.T) {
	dataStorage := NewCmapStorage()

	key := "key"

	if err := dataStorage.Delete(key); err == nil {
		t.Fatalf("delete operation did not fail")
	}

	dataStorage.values.Set(key, Data{[]byte{}, ""})
	err := dataStorage.Delete(key)
	if err != nil {
		t.Fatalf("delete operation failed")
	}
	if !dataStorage.values.IsEmpty() {
		t.Fatalf("storage not empty")
	}
}

func TestCmapStorage_Keys(t *testing.T) {
	dataStorage := NewCmapStorage()

	if len(dataStorage.Keys()) != 0 {
		t.Fatalf("storage keys count not equal to 0")
	}

	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		dataStorage.values.Set(key, Data{[]byte{}, ""})
	}

	extractedKeys := dataStorage.Keys()
	sort.Strings(keys)
	sort.Strings(extractedKeys)
	if !reflect.DeepEqual(keys, extractedKeys) {
		t.Fatalf("extracted keys differ: %v", extractedKeys)
	}
}
