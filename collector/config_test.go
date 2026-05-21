package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads from env file", func(t *testing.T) {
		t.Setenv("SSTATS_API_KEY", "")
		t.Setenv("SSTATS_BASE_URL", "")

		tempDir := t.TempDir()
		envPath := filepath.Join(tempDir, ".env")

		envContent := "SSTATS_API_KEY=test-api-key\nSSTATS_BASE_URL=https://api.test.com\n"
		if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tempDir)

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.APIKey != "test-api-key" {
			t.Errorf("expected APIKey 'test-api-key', got '%s'", config.APIKey)
		}

		if config.BaseURL != "https://api.test.com" {
			t.Errorf("expected BaseURL 'https://api.test.com', got '%s'", config.BaseURL)
		}
	})

	t.Run("missing api key", func(t *testing.T) {
		t.Setenv("SSTATS_API_KEY", "")
		t.Setenv("SSTATS_BASE_URL", "")

		tempDir := t.TempDir()
		envPath := filepath.Join(tempDir, ".env")

		envContent := "SSTATS_BASE_URL=https://api.test.com\n"
		if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tempDir)

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for missing API key")
		}
	})

	t.Run("missing base url", func(t *testing.T) {
		t.Setenv("SSTATS_API_KEY", "")
		t.Setenv("SSTATS_BASE_URL", "")

		tempDir := t.TempDir()
		envPath := filepath.Join(tempDir, ".env")

		envContent := "SSTATS_API_KEY=test-api-key\n"
		if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tempDir)

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for missing base URL")
		}
	})

	t.Run("env file not found", func(t *testing.T) {
		tempDir := t.TempDir()

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tempDir)

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for missing .env file")
		}
	})
}

func TestFindProjectRoot(t *testing.T) {
	t.Run("finds root with .env", func(t *testing.T) {
		tempDir := t.TempDir()
		envPath := filepath.Join(tempDir, ".env")

		if err := os.WriteFile(envPath, []byte(""), 0o644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		subDir := filepath.Join(tempDir, "subdir", "nested")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(subDir)

		root, err := findProjectRoot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if root != tempDir {
			t.Errorf("expected root '%s', got '%s'", tempDir, root)
		}
	})

	t.Run("not found", func(t *testing.T) {
		tempDir := t.TempDir()

		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tempDir)

		_, err := findProjectRoot()
		if err == nil {
			t.Fatal("expected error when .env not found")
		}
	})
}

func TestParseEnvFile(t *testing.T) {
	t.Run("parses valid env file", func(t *testing.T) {
		tempDir := t.TempDir()
		envPath := filepath.Join(tempDir, ".env")

		envContent := `# Comment
KEY1=value1
KEY2 = value2
KEY3="quoted value"
KEY4='single quoted'
KEY5=

# Another comment
KEY6=final
`
		if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		values, err := parseEnvFile(envPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "quoted value",
			"KEY4": "single quoted",
			"KEY5": "",
			"KEY6": "final",
		}

		for key, expectedValue := range expected {
			if values[key] != expectedValue {
				t.Errorf("key %s: expected '%s', got '%s'", key, expectedValue, values[key])
			}
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := parseEnvFile("/nonexistent/path/.env")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tempDir := t.TempDir()
		envPath := filepath.Join(tempDir, ".env")

		if err := os.WriteFile(envPath, []byte(""), 0o644); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		values, err := parseEnvFile(envPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(values) != 0 {
			t.Errorf("expected empty map, got %d entries", len(values))
		}
	})
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{"first non-empty", []string{"", "  ", "value", "other"}, "value"},
		{"all empty", []string{"", "  ", ""}, ""},
		{"single value", []string{"only"}, "only"},
		{"empty args", []string{}, ""},
		{"trim spaces", []string{"  ", " trimmed ", "other"}, " trimmed "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstNonEmpty(tt.values...)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
