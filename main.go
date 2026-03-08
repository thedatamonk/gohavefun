package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rohil/gofun/handler"
	"github.com/rohil/gofun/store"
)

func main() {
	fs := store.NewFeatureStore()

	// Seed sample data
	fs.Set("user", "123", store.FeatureVector{"age": 25, "score": 0.85, "active_days": 120})
	fs.Set("user", "456", store.FeatureVector{"age": 32, "score": 0.72, "active_days": 45})
	fs.Set("item", "abc", store.FeatureVector{"price": 29.99, "popularity": 0.91})
	fs.Set("item", "def", store.FeatureVector{"price": 9.99, "popularity": 0.45})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler.New(fs),
	}

	// Start server in a goroutine
	go func() {
		fmt.Println("Feature store listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nShutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped.")
}
