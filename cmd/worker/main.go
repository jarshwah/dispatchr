package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dispatchr"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// RiverClient interface captures the methods we use from river.Client
type RiverClient interface {
	Stop(ctx context.Context) error
}

func main() {
	ctx := context.Background()
	worker := setupWorker(ctx)
	shutdownHandler(ctx, worker)
}

func setupWorker(ctx context.Context) *river.Client[pgx.Tx] {
	fmt.Fprintf(os.Stdout, "Starting worker... \n")
	dbPool, err := dispatchr.CreateDatabasePool(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	client, err := dispatchr.CreateClient(ctx, dbPool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create client: %v\n", err)
		os.Exit(1)
	}

	if err := client.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start client: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Client started\n")
	return client
}

func shutdownHandler(ctx context.Context, client RiverClient) {
	shutdownChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownChannel, syscall.SIGINT, syscall.SIGTERM)
	sig := <-shutdownChannel
	fmt.Fprintf(os.Stdout, "Received signal %v, shutting down\n", sig)

	// Create a timeout context for graceful shutdown
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := client.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Client stopped gracefully\n")
}
