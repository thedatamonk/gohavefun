package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/handler"
	"github.com/rohil/gofun/seed"
	"github.com/rohil/gofun/store"
)

func main() {
	fs := store.NewFeatureStore()

	// Generate seed data
	customerIDs := seed.Generate(fs, 75)
	fmt.Printf("Seeded %d customers\n", len(customerIDs))

	// Start materializer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mat := feature.NewMaterializer(fs, customerIDs)
	go mat.Start(ctx, 10*time.Second)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler.New(fs),
	}

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

	// Stop materializer
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped.")
}
