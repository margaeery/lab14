package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	client, err := NewAPIClient(config)
	if err != nil {
		return err
	}

	writer, err := NewJSONLinesWriter(config.OutputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	service := NewCollectorService(client, writer)

	if err := service.CollectLeagues(ctx); err != nil {
		return err
	}

	return nil
}