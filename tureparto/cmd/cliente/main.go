package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/doro098/tureparto/internal/database"
	"github.com/doro098/tureparto/internal/poller"
	"github.com/doro098/tureparto/internal/visor"
	"github.com/doro098/tureparto/internal/webhook"
)

func main() {
	sourcePath := os.Getenv("DB_SOURCE_PATH")
	if sourcePath == "" {
		sourcePath = "/data/readonly/tureparto.db"
	}
	richPath := os.Getenv("DB_RICH_PATH")
	if richPath == "" {
		richPath = "/data/rich/tureparto_rich.db"
	}
	muleURL := os.Getenv("TAG_MULE_URL")
	if muleURL == "" {
		muleURL = "http://tag-mule:8080/api/v1/enrich"
	}
	portStr := os.Getenv("CLIENTE_PORT")
	if portStr == "" {
		portStr = "3001"
	}
	intervalStr := os.Getenv("POLL_INTERVAL_SECONDS")
	interval := 30
	if v, err := strconv.Atoi(intervalStr); err == nil && v > 0 {
		interval = v
	}

	db, err := database.Open(sourcePath, richPath)
	if err != nil {
		log.Fatalf("failed to open databases: %v", err)
	}
	defer db.Close()

	// Motor 1: Poller
	p := poller.New(db, muleURL, interval)
	p.Start()

	// Motor 2: Webhook
	wh := webhook.NewHandler(db)

	// Motor 3: Visor
	v := visor.New(db)

	mux := http.NewServeMux()
	mux.Handle("/webhook", wh)
	mux.HandleFunc("/api/data", v.DataHandler)
	mux.Handle("/", http.FileServer(http.Dir("internal/visor/static")))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", portStr),
		Handler: mux,
	}

	log.Printf("cliente-tureparto listening on :%s", portStr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
