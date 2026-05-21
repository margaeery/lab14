package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAPIClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := Config{
			APIKey:  "test-key",
			BaseURL: "https://api.example.com",
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.apiKey != "test-key" {
			t.Errorf("expected apiKey 'test-key', got '%s'", client.apiKey)
		}

		if client.baseURL.Host != "api.example.com" {
			t.Errorf("expected host 'api.example.com', got '%s'", client.baseURL.Host)
		}
	})

	t.Run("invalid base url", func(t *testing.T) {
		config := Config{
			APIKey:  "test-key",
			BaseURL: "://invalid-url",
		}

		_, err := NewAPIClient(config)
		if err == nil {
			t.Fatal("expected error for invalid URL")
		}
	})
}

func TestAPIClient_FetchCountries(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedCountries := []Country{
			{ID: 1, Code: "RU", Name: "Russia", URL: "/country/ru"},
			{ID: 2, Code: "EN", Name: "England", URL: "/country/en"},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/Pari/countries" {
				t.Errorf("expected path '/Pari/countries', got '%s'", r.URL.Path)
			}

			apiKey := r.URL.Query().Get("apikey")
			if apiKey != "test-key" {
				t.Errorf("expected apikey 'test-key', got '%s'", apiKey)
			}

			response := apiResponse[Country]{
				Status: "ok",
				Data:   expectedCountries,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		countries, err := client.FetchCountries(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(countries) != len(expectedCountries) {
			t.Fatalf("expected %d countries, got %d", len(expectedCountries), len(countries))
		}

		for i, expected := range expectedCountries {
			if countries[i].ID != expected.ID {
				t.Errorf("country %d: expected ID %d, got %d", i, expected.ID, countries[i].ID)
			}
			if countries[i].Code != expected.Code {
				t.Errorf("country %d: expected Code %s, got %s", i, expected.Code, countries[i].Code)
			}
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = client.FetchCountries(context.Background())
		if err == nil {
			t.Fatal("expected error for server error")
		}
	})

	t.Run("invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = client.FetchCountries(context.Background())
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("api error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := apiResponse[Country]{
				Status:  "error",
				Message: "API error message",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = client.FetchCountries(context.Background())
		if err == nil {
			t.Fatal("expected error for API error status")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = client.FetchCountries(ctx)
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
	})
}

func TestAPIClient_FetchLeaguesPage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedLeagues := []League{
			{ID: 10, Name: "Premier League", URL: "/league/10"},
			{ID: 11, Name: "Championship", URL: "/league/11"},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/Pari/leagues" {
				t.Errorf("expected path '/Pari/leagues', got '%s'", r.URL.Path)
			}

			query := r.URL.Query()
			if query.Get("countryId") != "5" {
				t.Errorf("expected countryId '5', got '%s'", query.Get("countryId"))
			}
			if query.Get("offset") != "10" {
				t.Errorf("expected offset '10', got '%s'", query.Get("offset"))
			}
			if query.Get("limit") != "20" {
				t.Errorf("expected limit '20', got '%s'", query.Get("limit"))
			}

			totalCount := 100
			response := apiResponse[League]{
				Status:     "ok",
				Data:       expectedLeagues,
				TotalCount: &totalCount,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		response, err := client.FetchLeaguesPage(context.Background(), 5, 10, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(response.Data) != len(expectedLeagues) {
			t.Fatalf("expected %d leagues, got %d", len(expectedLeagues), len(response.Data))
		}

		if response.TotalCount == nil || *response.TotalCount != 100 {
			t.Errorf("expected TotalCount 100, got %v", response.TotalCount)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		requestCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			response := apiResponse[League]{
				Status: "ok",
				Data:   []League{{ID: 1, Name: "League"}},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = client.FetchLeaguesPage(context.Background(), 1, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error after retry: %v", err)
		}

		if requestCount != 2 {
			t.Errorf("expected 2 requests (1 failed + 1 retry), got %d", requestCount)
		}
	})
}

func TestDoRequest_Retries(t *testing.T) {
	t.Run("retries on server error then succeeds", func(t *testing.T) {
		requestCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := apiResponse[Country]{
				Status: "ok",
				Data:   []Country{{ID: 1, Name: "Country"}},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = client.FetchCountries(context.Background())
		if err != nil {
			t.Fatalf("unexpected error after retries: %v", err)
		}

		if requestCount != 3 {
			t.Errorf("expected 3 requests, got %d", requestCount)
		}
	})

	t.Run("fails after max retries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, err := NewAPIClient(config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = client.FetchCountries(context.Background())
		if err == nil {
			t.Fatal("expected error after max retries")
		}
	})
}
