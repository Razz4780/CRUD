package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/storage"
	"github.com/go-chi/chi"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
)

func requestWithKey(r *http.Request, key string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", key)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func failingHandler(t *testing.T) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request passed middleware")
	})
}

func simpleHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func prepopulatedStorage(keys []string) storage.Storage {
	dataStorage := storage.NewStorage()
	for _, key := range keys {
		dataStorage.Put(key, []byte{}, "")
	}
	return dataStorage
}

func assertCodesEqual(t *testing.T, w *httptest.ResponseRecorder, code int) {
	if w.Code != code {
		t.Errorf("wrong response code: %v", w.Code)
	}
}

func assertBodiesEqual(t *testing.T, w *httptest.ResponseRecorder, body []byte) {
	responseBody := w.Body.Bytes()
	if !bytes.Equal(responseBody, body) {
		t.Errorf("wrong body: %v", body)
	}
}

func assertContentTypeEqual(t *testing.T, w *httptest.ResponseRecorder, contentType string) {
	responseContentType := w.Header().Get("Content-Type")
	if responseContentType != contentType {
		t.Errorf("wrong content type: %v", responseContentType)
	}
}

func assertBodyEmpty(t *testing.T, w *httptest.ResponseRecorder) {
	assertBodiesEqual(t, w, []byte{})
}

func TestCheckKey(t *testing.T) {
	allValidChars := "1234567890qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
	ofLength1 := "A"
	ofLength100Runes := make([]rune, 100)
	for i := range ofLength100Runes {
		ofLength100Runes[i] = 65
	}
	ofLength100 := string(ofLength100Runes)
	for _, key := range []string{allValidChars, ofLength1, ofLength100} {
		t.Run(fmt.Sprintf("valid: %v", key), func(t *testing.T) {
			w := httptest.NewRecorder()
			r := requestWithKey(httptest.NewRequest("", "/", nil), key)
			handler := checkKey(simpleHandler())
			handler.ServeHTTP(w, r)
			assertCodesEqual(t, w, http.StatusOK)
		})
	}

	ofLength0 := ""
	invalidChar := "-"
	for _, key := range []string{ofLength0, invalidChar} {
		t.Run(fmt.Sprintf("invalid: %v", key), func(t *testing.T) {
			w := httptest.NewRecorder()
			r := requestWithKey(httptest.NewRequest("", "/", nil), key)
			handler := checkKey(failingHandler(t))
			handler.ServeHTTP(w, r)
			assertCodesEqual(t, w, http.StatusBadRequest)
		})
	}

	t.Run("no key", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("", "/", nil)
		rctx := chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		handler := checkKey(failingHandler(t))
		handler.ServeHTTP(w, r)
		assertCodesEqual(t, w, http.StatusBadRequest)
	})
}

func TestRequireContentTypeHeader(t *testing.T) {
	t.Run("with header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("", "/", nil)
		r.Header.Set("Content-Type", "application/json")
		handler := requireContentTypeHeader(simpleHandler())
		handler.ServeHTTP(w, r)
		assertCodesEqual(t, w, http.StatusOK)
	})

	t.Run("no header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("", "/", nil)
		handler := requireContentTypeHeader(simpleHandler())
		handler.ServeHTTP(w, r)
		assertCodesEqual(t, w, http.StatusBadRequest)
	})
}

func TestLimitBodySize(t *testing.T) {
	t.Run("no body change", func(t *testing.T) {
		w := httptest.NewRecorder()
		originalBody := make([]byte, MaxObjectSize)
		buff := bytes.NewBuffer(originalBody)
		r := httptest.NewRequest("", "/", buff)
		handler := limitBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if !bytes.Equal(originalBody, body) {
				t.Error("body differs")
			}
		}))
		handler.ServeHTTP(w, r)
	})

	t.Run("body limited", func(t *testing.T) {
		w := httptest.NewRecorder()
		originalBody := make([]byte, MaxObjectSize+1)
		buff := bytes.NewBuffer(originalBody)
		r := httptest.NewRequest("", "/", buff)
		handler := limitBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			if err == nil {
				t.Error("no error")
			}
			if len(body) > MaxObjectSize {
				t.Errorf("body too large: %v", len(body))
			}
		}))
		handler.ServeHTTP(w, r)
	})
}

func TestPutObject(t *testing.T) {
	dataSets := []struct {
		keys        []string
		key         string
		object      []byte
		contentType string
	}{
		{[]string{"key1", "key2", "key3"}, "key", []byte{0, 1, 2, 3, 3}, "type"},
		{[]string{}, "key", []byte{0, 0, 0, 0}, "type2"},
		{[]string{"key"}, "key", []byte{1, 2, 3}, "type3"},
	}

	for i, dataSet := range dataSets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(dataSet.keys)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("", "/", bytes.NewBuffer(dataSet.object))
			r = requestWithKey(r, dataSet.key)
			r.Header.Set("Content-Type", dataSet.contentType)
			handler := putObject(dataStorage)
			handler.ServeHTTP(w, r)

			assertCodesEqual(t, w, http.StatusCreated)
			assertBodyEmpty(t, w)

			if data, err := dataStorage.Get(dataSet.key); err != nil {
				t.Error("key not in storage")
			} else {
				if data.ContentType != dataSet.contentType {
					t.Errorf("stored content type differs: %v", data.ContentType)
				}
				if !bytes.Equal(data.Object, dataSet.object) {
					t.Errorf("stored object differs: %v", data.Object)
				}
			}
		})
	}
}

func TestGetObject(t *testing.T) {
	dataSets := []struct {
		keys        []string
		searchedKey string
		object      []byte
		contentType string
	}{
		{[]string{"key"}, "key", []byte{0, 1, 2, 3, 4}, "type1"},
		{[]string{"key1", "key2", "key3"}, "key", []byte{}, ""},
		{[]string{"0", "1", "2", "3", "4"}, "5", []byte{}, ""},
		{[]string{"alpha", "beta"}, "alpha", []byte{}, "typeAlpha"},
	}

	for i, dataSet := range dataSets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(dataSet.keys)
			_, err := dataStorage.Get(dataSet.searchedKey)
			inStorage := err == nil
			if inStorage {
				dataStorage.Put(dataSet.searchedKey, dataSet.object, dataSet.contentType)
			}

			handler := getObject(dataStorage)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("", "/", nil)
			r = requestWithKey(r, dataSet.searchedKey)
			handler.ServeHTTP(w, r)

			if inStorage {
				assertCodesEqual(t, w, http.StatusOK)
				assertContentTypeEqual(t, w, dataSet.contentType)
				assertBodiesEqual(t, w, dataSet.object)
			} else {
				assertCodesEqual(t, w, http.StatusNotFound)
			}
		})
	}
}

func TestDeleteObject(t *testing.T) {
	dataSets := []struct {
		keys    []string
		deleted string
	}{
		{[]string{"key"}, "key"},
		{[]string{"key1", "key2", "key3"}, "key"},
		{[]string{}, "aa"},
		{[]string{"1", "2", "3", "4", "5"}, "1"},
	}

	for i, dataSet := range dataSets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(dataSet.keys)
			_, err := dataStorage.Get(dataSet.deleted)
			inStorage := err == nil

			w := httptest.NewRecorder()
			r := httptest.NewRequest("", "/", nil)
			r = requestWithKey(r, dataSet.deleted)
			handler := deleteObject(dataStorage)
			handler.ServeHTTP(w, r)

			if inStorage {
				assertCodesEqual(t, w, http.StatusNoContent)
				assertBodyEmpty(t, w)
				if _, err := dataStorage.Get(dataSet.deleted); err == nil {
					t.Errorf("deleted key in storage")
				}
				if !(len(dataStorage.Keys()) < len(dataSet.keys)) {
					t.Errorf("invalid storage state: %v", dataStorage.Keys())
				}
			} else {
				assertCodesEqual(t, w, http.StatusNotFound)
				assertBodyEmpty(t, w)
				if len(dataStorage.Keys()) != len(dataSet.keys) {
					t.Errorf("invalid storage state: %v", dataStorage.Keys())
				}
			}
		})
	}
}

func TestGetAllObjects(t *testing.T) {
	keySets := [][]string{
		{},
		{"key"},
		{"alpha", "beta", "gamma"},
		{"0", "1", "2", "3"},
	}
	for i, keySet := range keySets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(keySet)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("", "/", nil)
			handler := getAllObjects(dataStorage)
			handler.ServeHTTP(w, r)

			assertCodesEqual(t, w, http.StatusOK)
			var responseKeys []string
			if err := json.Unmarshal(w.Body.Bytes(), &responseKeys); err != nil {
				t.Fatal(err)
			}
			sort.Strings(keySet)
			sort.Strings(responseKeys)
			if !reflect.DeepEqual(keySet, responseKeys) {
				t.Errorf("keys differ: %v", responseKeys)
			}
		})
	}
}

func TestEndpointPut(t *testing.T) {
	dataSets := []struct {
		keys        []string
		key         string
		object      []byte
		contentType string
	}{
		{[]string{}, "key", []byte{0, 1, 2, 3}, "application/json"},
		{[]string{"key"}, "key", []byte{1, 1, 1, 1, 1}, "plain"},
		{[]string{"0", "1", "2", "3"}, "0", []byte{34, 12, 0, 23}, "0"},
		{[]string{}, "key", make([]byte, MaxObjectSize), "big"},
		{[]string{}, "key", make([]byte, MaxObjectSize+1), "too big"},
	}

	for i, dataSet := range dataSets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(dataSet.keys)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("PUT", ObjectsUrl+"/"+dataSet.key, bytes.NewBuffer(dataSet.object))
			r.Header.Set("Content-Type", dataSet.contentType)
			handler := NewRouter(dataStorage)
			handler.ServeHTTP(w, r)

			if len(dataSet.object) > MaxObjectSize {
				assertCodesEqual(t, w, http.StatusRequestEntityTooLarge)
				assertBodyEmpty(t, w)
				if _, err := dataStorage.Get(dataSet.key); err == nil {
					t.Errorf("object added to storage")
				}
			} else {
				assertCodesEqual(t, w, http.StatusCreated)
				assertBodyEmpty(t, w)
				if val, err := dataStorage.Get(dataSet.key); err != nil {
					t.Errorf("object not added to storage")
				} else {
					if val.ContentType != dataSet.contentType {
						t.Errorf("wrong content type: %v", val.ContentType)
					}
					if !bytes.Equal(val.Object, dataSet.object) {
						t.Errorf("wrong object: %v", val.Object)
					}
				}
			}
		})
	}

	t.Run("invalid key", func(t *testing.T) {
		dataStorage := storage.NewStorage()

		w := httptest.NewRecorder()
		r := httptest.NewRequest("PUT", ObjectsUrl+"/abc---", bytes.NewBuffer([]byte{}))
		r.Header.Set("Content-Type", "type")
		handler := NewRouter(dataStorage)
		handler.ServeHTTP(w, r)

		assertCodesEqual(t, w, http.StatusBadRequest)
		assertBodyEmpty(t, w)
	})

	t.Run("no Content-Type header", func(t *testing.T) {
		dataStorage := storage.NewStorage()

		w := httptest.NewRecorder()
		r := httptest.NewRequest("PUT", ObjectsUrl+"/abc", bytes.NewBuffer([]byte{}))
		handler := NewRouter(dataStorage)
		handler.ServeHTTP(w, r)

		assertCodesEqual(t, w, http.StatusBadRequest)
		assertBodyEmpty(t, w)
	})
}

func TestEndpointGet(t *testing.T) {
	dataSetsKeyPresent := []struct {
		keys        []string
		key         string
		object      []byte
		contentType string
	}{
		{[]string{"key1", "key2"}, "key3", []byte{1, 4, 12, 4}, "type"},
		{[]string{}, "key", []byte{0, 0, 0}, "type2"},
	}

	for i, dataSet := range dataSetsKeyPresent {
		t.Run(fmt.Sprint("present ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(dataSet.keys)
			dataStorage.Put(dataSet.key, dataSet.object, dataSet.contentType)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", ObjectsUrl+"/"+dataSet.key, nil)
			handler := NewRouter(dataStorage)
			handler.ServeHTTP(w, r)

			assertCodesEqual(t, w, http.StatusOK)
			assertBodiesEqual(t, w, dataSet.object)
			assertContentTypeEqual(t, w, dataSet.contentType)
		})
	}

	dataSetsKeyNotPresent := []struct {
		keys []string
		key  string
	}{
		{[]string{"key1", "key2"}, "key3"},
		{[]string{}, "key"},
	}

	for i, dataSet := range dataSetsKeyNotPresent {
		t.Run(fmt.Sprint("not present ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(dataSet.keys)

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", ObjectsUrl+"/"+dataSet.key, nil)
			handler := NewRouter(dataStorage)
			handler.ServeHTTP(w, r)

			assertCodesEqual(t, w, http.StatusNotFound)
			assertBodyEmpty(t, w)
		})
	}

	t.Run("invalid key", func(t *testing.T) {
		dataStorage := storage.NewStorage()

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", ObjectsUrl+"/abc---", nil)
		handler := NewRouter(dataStorage)
		handler.ServeHTTP(w, r)

		assertCodesEqual(t, w, http.StatusBadRequest)
		assertBodyEmpty(t, w)
	})
}

func TestEndpointDelete(t *testing.T) {
	dataSets := []struct {
		keys []string
		key  string
	}{
		{[]string{"0", "1", "2", "3"}, "0"},
		{[]string{}, "key"},
		{[]string{"key"}, "key"},
	}

	for i, dataSet := range dataSets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(dataSet.keys)
			_, err := dataStorage.Get(dataSet.key)
			inStorage := err == nil

			w := httptest.NewRecorder()
			r := httptest.NewRequest("DELETE", ObjectsUrl+"/"+dataSet.key, nil)
			handler := NewRouter(dataStorage)
			handler.ServeHTTP(w, r)

			if inStorage {
				assertCodesEqual(t, w, http.StatusNoContent)
				assertBodyEmpty(t, w)
			} else {
				assertCodesEqual(t, w, http.StatusNotFound)
				assertBodyEmpty(t, w)
			}
		})
	}

	t.Run("invalid key", func(t *testing.T) {
		dataStorage := storage.NewStorage()

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", ObjectsUrl+"/abc---", nil)
		handler := NewRouter(dataStorage)
		handler.ServeHTTP(w, r)

		assertCodesEqual(t, w, http.StatusBadRequest)
		assertBodyEmpty(t, w)
	})
}

func TestEndpointGetAll(t *testing.T) {
	keySets := [][]string{
		{},
		{"key1", "key2", "key3"},
		{"key"},
	}

	for i, keySet := range keySets {
		t.Run(fmt.Sprint("set ", i), func(t *testing.T) {
			dataStorage := prepopulatedStorage(keySet)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", ObjectsUrl, nil)
			handler := NewRouter(dataStorage)
			handler.ServeHTTP(w, r)

			assertCodesEqual(t, w, http.StatusOK)
			var responseKeys []string
			if err := json.Unmarshal(w.Body.Bytes(), &responseKeys); err != nil {
				t.Fatal(err)
			}
			sort.Strings(keySet)
			sort.Strings(responseKeys)
			if !reflect.DeepEqual(keySet, responseKeys) {
				t.Errorf("keys differ: %v", responseKeys)
			}
		})
	}
}
