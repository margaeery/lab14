package main

import (
	"context"
	"fmt"
	"sync"
)

type CollectorService struct {
	client *APIClient
	writer *JSONLinesWriter
	limit  int
}

func NewCollectorService(client *APIClient, writer *JSONLinesWriter) *CollectorService {
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

	leagues := make(chan League)
	errCh := make(chan error, len(countries))

	var writerGroup sync.WaitGroup
	writerGroup.Add(1)
	go func() {
		defer writerGroup.Done()
		for league := range leagues {
			if err := service.writer.Write(league); err != nil {
				select {
				case errCh <- fmt.Errorf("write league %d: %w", league.ID, err):
				default:
				}
				cancel()
				return
			}
		}
	}()

	var workerGroup sync.WaitGroup
	for _, country := range countries {
		country := country
		workerGroup.Add(1)
		go func() {
			defer workerGroup.Done()
			if err := service.collectCountryLeagues(ctx, country.ID, leagues); err != nil {
				select {
				case errCh <- fmt.Errorf("collect leagues for country %d: %w", country.ID, err):
				default:
				}
				cancel()
			}
		}()
	}

	workerGroup.Wait()
	close(leagues)
	writerGroup.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (service *CollectorService) collectCountryLeagues(ctx context.Context, countryID int64, output chan<- League) error {
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
			case output <- league:
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