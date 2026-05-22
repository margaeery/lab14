package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type CollectorService struct {
	client *APIClient
	writer Writer
	limit  int
}

func NewCollectorService(client *APIClient, writer Writer) *CollectorService {
	return &CollectorService{
		client: client,
		writer: writer,
		limit:  1000,
	}
}

func (service *CollectorService) CollectLeagues(ctx context.Context) error {
	countries, err := service.client.FetchCountries(ctx)
	if err != nil {
		return fmt.Errorf("fetch countries: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	batchWriter := NewBatchWriter(service.writer, 50, 3*time.Second)
	batchWriter.Start()

	errCh := make(chan error, len(countries))

	var workerGroup sync.WaitGroup
	for _, country := range countries {
		country := country
		workerGroup.Add(1)
		go func() {
			defer workerGroup.Done()
			if err := service.collectCountryLeagues(ctx, country.ID, batchWriter); err != nil {
				select {
				case errCh <- fmt.Errorf("collect leagues for country %d: %w", country.ID, err):
				default:
				}
				cancel()
			}
		}()
	}

	workerGroup.Wait()
	closeErr := batchWriter.Close()

	select {
	case err := <-errCh:
		return err
	default:
		if closeErr != nil {
			return closeErr
		}
		return nil
	}
}

func (service *CollectorService) collectCountryLeagues(ctx context.Context, countryID int64, writer LeagueWriter) error {
	for offset := 0; ; offset += service.limit {
		if err := ctx.Err(); err != nil {
			return err
		}

		response, err := service.client.FetchLeaguesPage(ctx, countryID, offset, service.limit)
		if err != nil {
			return err
		}

		for _, league := range response.Data {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if err := writer.Write(league); err != nil {
					return err
				}
			}
		}

		totalCount := len(response.Data)
		if response.TotalCount != nil {
			totalCount = *response.TotalCount
		}

		if len(response.Data) == 0 || offset+len(response.Data) >= totalCount {
			return nil
		}
	}
}