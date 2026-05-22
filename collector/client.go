package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type APIClient struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	rateLimiter <-chan time.Time
}

func NewAPIClient(config Config) (*APIClient, error) {
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	return &APIClient{
		baseURL: baseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: time.NewTicker(250 * time.Millisecond).C,
	}, nil
}

func (client *APIClient) FetchCountries(ctx context.Context) ([]Country, error) {
	response, err := doRequest[Country](ctx, client, "/Pari/countries", nil)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (client *APIClient) FetchLeaguesPage(ctx context.Context, countryID int64, offset int, limit int) (apiResponse[League], error) {
	params := url.Values{}
	params.Set("countryId", strconv.FormatInt(countryID, 10))
	params.Set("offset", strconv.Itoa(offset))
	params.Set("limit", strconv.Itoa(limit))

	response, err := doRequest[League](ctx, client, "/Pari/leagues", params)
	if err != nil {
		return apiResponse[League]{}, err
	}

	return response, nil
}

func doRequest[T any](ctx context.Context, client *APIClient, endpoint string, params url.Values) (apiResponse[T], error) {
	requestURL := *client.baseURL
	requestURL.Path = path.Join(strings.TrimSuffix(client.baseURL.Path, "/"), endpoint)

	query := requestURL.Query()
	query.Set("apikey", client.apiKey)
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	requestURL.RawQuery = query.Encode()

	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if err := waitTurn(ctx, client.rateLimiter); err != nil {
			return apiResponse[T]{}, err
		}

		payload, retry, err := executeRequest[T](ctx, client, endpoint, requestURL.String())
		if err == nil {
			return payload, nil
		}

		lastErr = err
		if !retry {
			return apiResponse[T]{}, err
		}

		if err := sleepWithContext(ctx, time.Duration(attempt+1)*time.Second); err != nil {
			return apiResponse[T]{}, err
		}
	}

	return apiResponse[T]{}, lastErr
}

func executeRequest[T any](ctx context.Context, client *APIClient, endpoint string, requestURL string) (apiResponse[T], bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return apiResponse[T]{}, false, fmt.Errorf("create request %s: %w", endpoint, err)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return apiResponse[T]{}, true, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		return apiResponse[T]{}, true, fmt.Errorf("request %s returned status %d", endpoint, response.StatusCode)
	}

	if response.StatusCode != http.StatusOK {
		return apiResponse[T]{}, false, fmt.Errorf("request %s returned status %d", endpoint, response.StatusCode)
	}

	var payload apiResponse[T]
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return apiResponse[T]{}, false, fmt.Errorf("decode %s response: %w", endpoint, err)
	}

	validStatuses := map[string]bool{"ok": true, "success": true, "": true}
	if !validStatuses[strings.ToLower(payload.Status)] {
		return apiResponse[T]{}, false, fmt.Errorf("request %s failed: %s", endpoint, payload.Message)
	}

	return payload, false, nil
}

func waitTurn(ctx context.Context, limiter <-chan time.Time) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-limiter:
		return nil
	}
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}