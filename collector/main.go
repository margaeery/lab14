package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
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

	if err := service.CollectLeagues(context.Background()); err != nil {
		return err
	}

	return nil
}