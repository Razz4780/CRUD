package router

import (
	"encoding/json"
	"github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/storage"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"io/ioutil"
	"net/http"
	"regexp"
)

const (
	KeyPattern    = "^[0-9a-zA-Z]{1,100}$" // pattern describing valid keys
	MaxObjectSize = 1000000                // in bytes
	ObjectsUrl    = "/api/objects"
)

func NewRouter(dataStorage storage.Storage) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route(ObjectsUrl, func(router chi.Router) {
		router.Get("/", getAllObjects(dataStorage))
		router.Route("/{key}", func(router chi.Router) {
			router.Use(checkKey)
			router.With(
				requireContentTypeHeader,
				limitBodySize,
			).Put("/", putObject(dataStorage))
			router.Get("/", getObject(dataStorage))
			router.Delete("/", deleteObject(dataStorage))
		})
	})

	return router
}

// checkKey stops requests without valid key parameter,
// writing code http.StatusBadRequest.
func checkKey(next http.Handler) http.Handler {
	regex := regexp.MustCompile(KeyPattern)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")
		if match := regex.MatchString(key); match {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
}

// requireContentTypeHeader stops requests without Content-Type header,
// writing code http.StatusBadRequest.
func requireContentTypeHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct == "" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// limitBodySize limits request's body size to MaxObjectsize.
// Next handlers will receive an error when trying to read
// more then MaxObjectSize from request's body.
func limitBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxObjectSize)
		next.ServeHTTP(w, r)
	})
}

// putObject(storage) places request's body Content-Type header
// in storage under request's key parameter.
// If request's body is too big, writes code http.StatusRequestEntityTooLarge.
// Writes code http.StatusCreated otherwise.
func putObject(dataStorage storage.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if object, err := ioutil.ReadAll(r.Body); err == nil {
			key := chi.URLParam(r, "key")
			contentType := r.Header.Get("Content-Type")
			dataStorage.Put(key, object, contentType)
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}
	})
}

// getObject(storage) retrieves data stored in storage
// under request's key parameter.
// On successful retrieve, writes Object part of the data
// into body and sets Content-Type header to
// ContentType part of the data.
// Writes code http.StatusNotFound otherwise.
func getObject(dataStorage storage.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")
		if val, err := dataStorage.Get(key); err == nil {
			w.Header().Set("Content-Type", val.ContentType)
			if _, err := w.Write(val.Object); err != nil {
				panic(err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// deleteObject(storage) deletes data stored in storage
// under request's key parameter.
// On successful delete, writes code http.StatusNoContent.
// Writes code http.StatusNotFound otherwise.
func deleteObject(dataStorage storage.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")
		if err := dataStorage.Delete(key); err == nil {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// getAllObjects(storage) writes keys present in storage into
// body in JSON format.
func getAllObjects(dataStorage storage.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if keys, err := json.Marshal(dataStorage.Keys()); err == nil {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(keys); err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	})
}
