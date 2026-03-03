package main

import (
	"log"
	"net/http"
	"os"

	"claude-bot/internal/api"
	"claude-bot/internal/claude"
	"claude-bot/internal/db"
	"claude-bot/internal/workflow"
	"claude-bot/internal/ws"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "claude-bot.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	queries := db.NewQueries(database)
	hub := ws.NewHub()
	go hub.Run()

	runner := claude.NewRunner(queries)
	engine := workflow.NewEngine(queries, runner, hub)

	srv := &api.Server{
		DB:      database,
		Queries: queries,
		Hub:     hub,
		Engine:  engine,
		Runner:  runner,
	}

	router := api.NewRouter(srv)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
