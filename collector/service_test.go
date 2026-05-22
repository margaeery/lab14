package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewCollectorService(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test.jsonl")

	config := Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
	}

	client, _ := NewAPIClient(config)
	writer, err := NewJSONLinesWriter(outputPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer writer.Close()

	service := NewCollectorService(client, writer)

	if service.client != client {
		t.Error("client not set correctly")
	}

	if service.writer != writer {
		t.Error("writer not set correctly")
	}

	if service.limit != 1000 {
		t.Errorf("expected limit 1000, got %d", service.limit)
	}
}

func TestCollectorService_CollectLeagues(t *testing.T) {
	t.Run("success - collects all leagues", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		countriesResponse := apiResponse[Country]{
			Status: "ok",
			Data: []Country{
				{ID: 1, Code: "RU", Name: "Russia"},
				{ID: 2, Code: "EN", Name: "England"},
			},
		}

		leaguesByCountry := map[int64][]League{
			1: {
				{ID: 10, Name: "RPL", URL: "/league/10"},
				{ID: 11, Name: "FNL", URL: "/league/11"},
			},
			2: {
				{ID: 20, Name: "Premier League", URL: "/league/20"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.URL.Path == "/Pari/countries" {
				json.NewEncoder(w).Encode(countriesResponse)
				return
			}

			if r.URL.Path == "/Pari/leagues" {
				countryID := r.URL.Query().Get("countryId")
				var cid int64
				fmt.Sscanf(countryID, "%d", &cid)

				leagues := leaguesByCountry[cid]
				totalCount := len(leagues)

				response := apiResponse[League]{
					Status:     "ok",
					Data:       leagues,
					TotalCount: &totalCount,
				}

				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
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

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		service := NewCollectorService(client, writer)

		ctx := context.Background()
		err = service.CollectLeagues(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		writer.Close()

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		expectedCount := 3
		if len(lines) != expectedCount {
			t.Errorf("expected %d leagues, got %d", expectedCount, len(lines))
		}
	})

	t.Run("empty countries", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.URL.Path == "/Pari/countries" {
				response := apiResponse[Country]{
					Status: "ok",
					Data:   []Country{},
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
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

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		service := NewCollectorService(client, writer)

		ctx := context.Background()
		err = service.CollectLeagues(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		writer.Close()

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		if strings.TrimSpace(string(content)) != "" {
			t.Errorf("expected empty output for empty countries")
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

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		service := NewCollectorService(client, writer)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = service.CollectLeagues(ctx)
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
	})

	t.Run("fetch countries error", func(t *testing.T) {
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

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		service := NewCollectorService(client, writer)

		ctx := context.Background()
		err = service.CollectLeagues(ctx)
		if err == nil {
			t.Fatal("expected error for fetch countries failure")
		}

		if !strings.Contains(err.Error(), "fetch countries") {
			t.Errorf("expected 'fetch countries' in error, got '%v'", err)
		}
	})
}

type channelLeagueWriter struct {
	ch chan League
}

func (w *channelLeagueWriter) Write(league League) error {
	w.ch <- league
	return nil
}

func TestCollectorService_collectCountryLeagues(t *testing.T) {
	t.Run("single page", func(t *testing.T) {
		leagues := []League{
			{ID: 1, Name: "League 1"},
			{ID: 2, Name: "League 2"},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			totalCount := len(leagues)
			response := apiResponse[League]{
				Status:     "ok",
				Data:       leagues,
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

		client, _ := NewAPIClient(config)

		service := NewCollectorService(client, &JSONLinesWriter{})
		output := make(chan League, 10)
		writer := &channelLeagueWriter{ch: output}

		err := service.collectCountryLeagues(context.Background(), 1, writer)
		close(output)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collected := make([]League, 0, len(output))
		for league := range output {
			collected = append(collected, league)
		}

		if len(collected) != len(leagues) {
			t.Errorf("expected %d leagues, got %d", len(leagues), len(collected))
		}
	})

	t.Run("multiple pages", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		page1 := []League{{ID: 1, Name: "League 1"}, {ID: 2, Name: "League 2"}}
		page2 := []League{{ID: 3, Name: "League 3"}}

		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			totalCount := 3

			var data []League
			if requestCount == 1 {
				data = page1
			} else {
				data = page2
			}

			response := apiResponse[League]{
				Status:     "ok",
				Data:       data,
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

		client, _ := NewAPIClient(config)

		jsonWriter, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer jsonWriter.Close()

		service := &CollectorService{
			client: client,
			writer: jsonWriter,
			limit:  2,
		}

		output := make(chan League, 10)
		writer := &channelLeagueWriter{ch: output}

		err = service.collectCountryLeagues(context.Background(), 1, writer)
		close(output)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collected := make([]League, 0, len(output))
		for league := range output {
			collected = append(collected, league)
		}

		if len(collected) != 3 {
			t.Errorf("expected 3 leagues, got %d", len(collected))
		}

		if requestCount != 2 {
			t.Errorf("expected 2 requests, got %d", requestCount)
		}
	})

	t.Run("context cancelled during collection", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
		}))
		defer server.Close()

		config := Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		}

		client, _ := NewAPIClient(config)

		jsonWriter, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer jsonWriter.Close()

		service := NewCollectorService(client, jsonWriter)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		output := make(chan League, 10)
		writer := &channelLeagueWriter{ch: output}
		err = service.collectCountryLeagues(ctx, 1, writer)
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
	})
}

func TestCollectorService_Integration(t *testing.T) {
	t.Run("full collection flow", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		countriesData := []Country{
			{ID: 1, Code: "RU", Name: "Russia"},
			{ID: 2, Code: "ES", Name: "Spain"},
		}

		leaguesData := []League{
			{ID: 10, Name: "RPL", Country: &countriesData[0]},
			{ID: 11, Name: "La Liga", Country: &countriesData[1]},
			{ID: 12, Name: "Segunda", Country: &countriesData[1]},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.URL.Path == "/Pari/countries" {
				response := apiResponse[Country]{
					Status: "ok",
					Data:   countriesData,
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			if r.URL.Path == "/Pari/leagues" {
				countryID := r.URL.Query().Get("countryId")
				var cid int64
				fmt.Sscanf(countryID, "%d", &cid)

				var filtered []League
				for _, league := range leaguesData {
					if league.Country != nil && league.Country.ID == cid {
						filtered = append(filtered, league)
					}
				}

				totalCount := len(filtered)
				response := apiResponse[League]{
					Status:     "ok",
					Data:       filtered,
					TotalCount: &totalCount,
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
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

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		service := NewCollectorService(client, writer)

		ctx := context.Background()
		err = service.CollectLeagues(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		writer.Close()

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != len(leaguesData) {
			t.Errorf("expected %d leagues, got %d", len(leaguesData), len(lines))
		}

		var decodedLeagues []League
		for _, line := range lines {
			var league League
			if err := json.Unmarshal([]byte(line), &league); err != nil {
				t.Fatalf("failed to decode line: %v", err)
			}
			decodedLeagues = append(decodedLeagues, league)
		}

		if len(decodedLeagues) != len(leaguesData) {
			t.Errorf("expected %d decoded leagues, got %d", len(leaguesData), len(decodedLeagues))
		}
	})

	t.Run("batch collection more than 50 records", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "leagues.jsonl")

		countriesData := []Country{
			{ID: 1, Code: "RU", Name: "Russia"},
		}

		leaguesData := make([]League, 55)
		for i := 0; i < 55; i++ {
			leaguesData[i] = League{ID: int64(i + 1), Name: fmt.Sprintf("League %d", i+1), Country: &countriesData[0]}
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if r.URL.Path == "/Pari/countries" {
				response := apiResponse[Country]{
					Status: "ok",
					Data:   countriesData,
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			if r.URL.Path == "/Pari/leagues" {
				totalCount := len(leaguesData)
				response := apiResponse[League]{
					Status:     "ok",
					Data:       leaguesData,
					TotalCount: &totalCount,
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
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

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		service := NewCollectorService(client, writer)

		ctx := context.Background()
		err = service.CollectLeagues(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		writer.Close()

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 55 {
			t.Errorf("expected 55 leagues, got %d", len(lines))
		}
	})
}