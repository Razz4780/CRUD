package persistence

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/router"
	"github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/storage"
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestSerialization(t *testing.T) {
	dataSet := []storage.Data{
		{[]byte{}, ""},
		{[]byte{1, 2, 3, 4}, "type"},
		{[]byte{}, "text"},
		{[]byte{4, 4, 4}, ""},
	}

	for i, data := range dataSet {
		t.Run(fmt.Sprint("data ", i), func(t *testing.T) {
			serialized := serializeData(data)
			deserialized, err := deserializeData(serialized)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data.Object, deserialized.Object) {
				t.Errorf("object differs: %v", deserialized.Object)
			}
			if data.ContentType != deserialized.ContentType {
				t.Errorf("content type differs: %v", deserialized.ContentType)
			}
		})
	}

	t.Run("invalid data deserialization", func(t *testing.T) {
		serialized := make([]byte, 4)
		binary.LittleEndian.PutUint16(serialized, 8)
		if _, err := deserializeData(serialized); err == nil {
			t.Error("deserialized invalid data")
		}
	})
}

func TestLoadSave(t *testing.T) {
	testDbName := "GWP_test.db"
	dataSets := [][]struct {
		key         string
		object      []byte
		contentType string
	}{
		{
			{"key", []byte{}, ""},
		},
		{
			{"key1", []byte{0, 2, 4, 1, 4}, "type1"},
			{"key2", []byte{4, 4, 4, 5, 3}, "type2"},
			{"key3", []byte{6, 2}, "type3"},
		},
		{
			{"0", make([]byte, router.MaxObjectSize), "big"},
		},
	}

	for i, dataSet := range dataSets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			originalStorage := storage.NewStorage()
			for _, values := range dataSet {
				originalStorage.Put(values.key, values.object, values.contentType)
			}
			if err := SaveToDb(originalStorage, testDbName); err != nil {
				t.Fatal(err)
			}
			loadedStorage := storage.NewStorage()
			if err := LoadFromDb(loadedStorage, testDbName); err != nil {
				t.Fatal(err)
			}

			originalKeys := originalStorage.Keys()
			loadedKeys := loadedStorage.Keys()
			sort.Strings(originalKeys)
			sort.Strings(loadedKeys)
			if !reflect.DeepEqual(originalKeys, loadedKeys) {
				t.Fatalf("keys differ: %v", loadedKeys)
			}
			for _, key := range originalKeys {
				originalData, _ := originalStorage.Get(key)
				loadedData, _ := loadedStorage.Get(key)
				if !reflect.DeepEqual(originalData, loadedData) {
					t.Errorf("data differs: %v", loadedData)
				}
			}
		})
	}

	if err := os.Remove(testDbName); err != nil {
		t.Fatal(err)
	}

	t.Run("empty db", func(t *testing.T) {
		dataStorage := storage.NewStorage()
		err := LoadFromDb(dataStorage, testDbName)
		if err == nil {
			t.Errorf("loaded data from empty db")
		}
		keys := dataStorage.Keys()
		if len(keys) > 0 {
			t.Errorf("invalid storage state: %v", keys)
		}
	})

	if err := os.Remove(testDbName); err != nil {
		t.Fatal(err)
	}
}
