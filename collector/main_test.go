package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRun_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiResponse[Country]{Status: "ok", Data: []Country{{ID: 1, Name: "Test"}}})
	}))
	defer server.Close()

	envContent := fmt.Sprintf("SSTATS_API_KEY=test\nSSTATS_BASE_URL=%s\n", server.URL)
	if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := run(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRun_ContextTimeout(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiResponse[Country]{Status: "ok", Data: []Country{{ID: 1, Name: "Test"}}})
	}))
	defer server.Close()

	envContent := fmt.Sprintf("SSTATS_API_KEY=test\nSSTATS_BASE_URL=%s\n", server.URL)
	if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := run(ctx)
	if err == nil {
		t.Fatal("expected error for timeout")
	}
}

func TestRun_Success(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	outputPath := filepath.Join(tempDir, "collector", "leagues_raw.json")

	countriesData := []Country{{ID: 1, Code: "RU", Name: "Russia"}}
	leaguesData := []League{{ID: 10, Name: "RPL"}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/Pari/countries" {
			json.NewEncoder(w).Encode(apiResponse[Country]{Status: "ok", Data: countriesData})
			return
		}
		if r.URL.Path == "/Pari/leagues" {
			totalCount := len(leaguesData)
			json.NewEncoder(w).Encode(apiResponse[League]{Status: "ok", Data: leaguesData, TotalCount: &totalCount})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	envContent := fmt.Sprintf("SSTATS_API_KEY=test\nSSTATS_BASE_URL=%s\n", server.URL)
	if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	ctx := context.Background()
	err := run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty output file")
	}
}
