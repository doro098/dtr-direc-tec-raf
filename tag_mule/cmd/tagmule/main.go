package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doro098/tag_mule/internal/api"
	"github.com/doro098/tag_mule/internal/config"
	"github.com/doro098/tag_mule/internal/storage"
	"github.com/doro098/tag_mule/internal/worker"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg := config.Load()

	db, err := storage.NewSQLite(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	pool := worker.NewPool(db, cfg, cfg.WorkerCount)
	pool.Start()

	handler := api.NewHandler(db, pool, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/enrich", handler.Enrich)
	mux.HandleFunc("GET /api/v1/health", handler.Health)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("tag-mule listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	pool.Shutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("shutdown complete")
}
