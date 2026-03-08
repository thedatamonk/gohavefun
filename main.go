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
	"github.com/rohil/gofun/registry"
	"github.com/rohil/gofun/scoring"
	"github.com/rohil/gofun/seed"
	"github.com/rohil/gofun/store"
)

func main() {
	os.MkdirAll("data", 0755)

	fs, err := store.NewSQLiteStore("data/gofun.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "store error: %v\n", err)
		os.Exit(1)
	}
	defer fs.Close()

	reg, err := registry.NewSQLiteRegistry("data/registry.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry error: %v\n", err)
		os.Exit(1)
	}
	defer reg.Close()

	// Seed registry and feature data
	seed.SeedRegistry(reg)
	customerIDs := seed.Generate(fs, 5000)
	fmt.Printf("Seeded %d customers\n", len(customerIDs))

	// Start materializer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mat := feature.NewMaterializer(fs, customerIDs)
	go mat.Start(ctx, 10*time.Second)

	scorer := scoring.NewScorer("models")

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler.New(fs, reg, scorer),
	}

	go func() {
		fmt.Println("Feature store listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nShutting down...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Server stopped.")
}
