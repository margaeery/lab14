package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewJSONLinesWriter(t *testing.T) {
	t.Run("creates file and directory", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "subdir", "output.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		if _, err := os.Stat(outputPath); err != nil {
			t.Errorf("output file was not created: %v", err)
		}
	})

	t.Run("invalid file name", func(t *testing.T) {
		_, err := NewJSONLinesWriter("")
		if err == nil {
			t.Fatal("expected error for empty path")
		}
	})
}

func TestJSONLinesWriter_Write(t *testing.T) {
	t.Run("writes valid json lines", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		league1 := League{ID: 1, Name: "Premier League", URL: "/league/1"}
		league2 := League{ID: 2, Name: "Championship", URL: "/league/2"}

		if err := writer.Write(league1); err != nil {
			t.Fatalf("unexpected error writing first line: %v", err)
		}

		if err := writer.Write(league2); err != nil {
			t.Fatalf("unexpected error writing second line: %v", err)
		}

		writer.Close()

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		var decoded1 League
		if err := json.Unmarshal([]byte(lines[0]), &decoded1); err != nil {
			t.Fatalf("failed to decode first line: %v", err)
		}

		if decoded1.ID != 1 || decoded1.Name != "Premier League" {
			t.Errorf("first line mismatch: got %+v", decoded1)
		}

		var decoded2 League
		if err := json.Unmarshal([]byte(lines[1]), &decoded2); err != nil {
			t.Fatalf("failed to decode second line: %v", err)
		}

		if decoded2.ID != 2 || decoded2.Name != "Championship" {
			t.Errorf("second line mismatch: got %+v", decoded2)
		}
	})

	t.Run("writes country", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		country := Country{ID: 10, Code: "RU", Name: "Russia", URL: "/country/ru"}

		if err := writer.Write(country); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		writer.Close()

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		var decoded Country
		if err := json.Unmarshal(content, &decoded); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if decoded.ID != 10 || decoded.Code != "RU" {
			t.Errorf("country mismatch: got %+v", decoded)
		}
	})
}

func TestJSONLinesWriter_Close(t *testing.T) {
	t.Run("close twice", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := writer.Close(); err != nil {
			t.Fatalf("unexpected error on first close: %v", err)
		}

		if err := writer.Close(); err != nil {
			t.Fatalf("unexpected error on second close: %v", err)
		}
	})
}

func TestJSONLinesWriter_Concurrent(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "output.jsonl")

	writer, err := NewJSONLinesWriter(outputPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer writer.Close()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			league := League{ID: int64(id), Name: "League", URL: "/league"}
			writer.Write(league)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	writer.Close()

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}
}

func TestJSONLinesWriter_Flush(t *testing.T) {
	t.Run("flush writes buffer to file", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer writer.Close()

		league := League{ID: 1, Name: "Test League"}
		if err := writer.Write(league); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := writer.Flush(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		if len(content) == 0 {
			t.Errorf("expected content after flush, got empty file")
		}

		var decoded League
		if err := json.Unmarshal(content, &decoded); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if decoded.ID != 1 {
			t.Errorf("expected ID 1, got %d", decoded.ID)
		}
	})

	t.Run("flush on closed writer", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonl")

		writer, err := NewJSONLinesWriter(outputPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		writer.Close()

		if err := writer.Flush(); err != nil {
			t.Fatalf("unexpected error on flush after close: %v", err)
		}
	})
}
