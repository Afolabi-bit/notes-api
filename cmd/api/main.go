package main

import (
	"fmt"
	"log"
	"notes-api/internal/config"
	"notes-api/internal/db"
	"notes-api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	client, _, err := db.Connect(cfg)

	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}

	defer func() {
		if err := db.Disconnect(client); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	router := server.NewRouter()

	addr := fmt.Sprintf(":%s", cfg.ServerPort)

	log.Printf("starting server on %s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
