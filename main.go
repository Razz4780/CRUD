package main

import (
	"context"
	"github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/persistence"
	GWPRouter "github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/router"
	"github.com/Razz4780/TWljaGHFgi1TbW9sYXJlaw/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	port            = ":8080"
	shutdownTimeout = 5
	dbName          = "gwp.db"
)

func main() {
	dataStorage := storage.NewStorage()
	if err := persistence.LoadFromDb(dataStorage, dbName); err != nil {
		log.Println(err)
		log.Println("Skipping loading data from db")
		dataStorage = storage.NewStorage()
	}

	server := &http.Server{Addr: port, Handler: GWPRouter.NewRouter(dataStorage)}

	go func() {
		// Here we catch SIGINT and SIGTERM signals
		// to shutdown the server and save the data.
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		log.Println("Shutting down the server...")
		ctx, _ := context.WithTimeout(context.Background(), shutdownTimeout*time.Second)
		if err := server.Shutdown(ctx); err != nil {
			log.Println(err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Println(err)
	}

	log.Println("Saving data...")
	if err := persistence.SaveToDb(dataStorage, dbName); err != nil {
		log.Fatal(err)
	}
}
