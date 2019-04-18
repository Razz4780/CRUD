package persistence

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/storage"
	"github.com/boltdb/bolt"
)

const (
	bucket = "GWP" // bucket name inside Bolt database
)

// LoadFromDb loads Bolt database contents to storage.
func LoadFromDb(dataStorage storage.Storage, dbName string) (rerr error) {
	if db, err := bolt.Open(dbName, 0600, nil); err != nil {
		return err
	} else {
		defer func() {
			err := db.Close()
			if rerr == nil {
				rerr = err
			}
		}()
		return db.View(func(tx *bolt.Tx) error {
			if gwp := tx.Bucket([]byte(bucket)); gwp != nil {
				return gwp.ForEach(func(k, v []byte) error {
					if data, err := deserializeData(v); err == nil {
						dataStorage.Put(string(k), data.Object, data.ContentType)
						return nil
					} else {
						// Unsuccessful deserialization means data inconsistency.
						return err
					}
				})
			} else {
				return errors.New(fmt.Sprintf("bucket %s not present", bucket))
			}
		})
	}
}

// SaveToDb saves storage contents to Bolt database.
func SaveToDb(dataStorage storage.Storage, dbName string) (rerr error) {
	if db, err := bolt.Open(dbName, 0600, nil); err != nil {
		return err
	} else {
		defer func() {
			err := db.Close()
			if rerr == nil {
				rerr = err
			}
		}()
		return db.Update(func(tx *bolt.Tx) error {
			_ = tx.DeleteBucket([]byte(bucket))
			if gwp, err := tx.CreateBucket([]byte(bucket)); err == nil {
				keys := dataStorage.Keys()
				for _, key := range keys {
					if data, err := dataStorage.Get(key); err == nil {
						if err := gwp.Put([]byte(key), serializeData(data)); err != nil {
							return err
						}
					} else {
						return err
					}
				}
				return nil
			} else {
				return err
			}
		})
	}
}

// serializeData serializes storage.Data struct into byte slice.
// First to bytes of slice contain ContentType as little endian uint16.
// Further bytes contain ContentType and Object.
func serializeData(data storage.Data) []byte {
	contentType := []byte(data.ContentType)
	contentTypeLen := len(contentType)
	serialized := make([]byte, 2+contentTypeLen+len(data.Object))
	binary.LittleEndian.PutUint16(serialized, uint16(contentTypeLen))
	copy(serialized[2:], contentType)
	copy(serialized[2+contentTypeLen:], data.Object)
	return serialized
}

// deserializeData deserializes byte slice into storage.Data struct.
// Deserializing struct serialized with serializeData will always
// be successful.
// deserializeData returns error on failure.
func deserializeData(serialized []byte) (storage.Data, error) {
	contentTypeLen := int(binary.LittleEndian.Uint16(serialized))
	if len(serialized) < 2+contentTypeLen {
		return storage.Data{}, errors.New("deseralization: invalid data")
	} else {
		contentType := string(serialized[2 : 2+contentTypeLen])
		object := make([]byte, len(serialized)-2-contentTypeLen)
		copy(object, serialized[2+contentTypeLen:])
		return storage.Data{Object: object, ContentType: contentType}, nil
	}
}
