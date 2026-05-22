package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey    string
	BaseURL   string
	RootDir   string
	OutputPath string
}

func LoadConfig() (Config, error) {
	rootDir, err := findProjectRoot()
	if err != nil {
		return Config{}, err
	}

	envPath := filepath.Join(rootDir, ".env")
	values, err := parseEnvFile(envPath)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		APIKey:    firstNonEmpty(os.Getenv("SSTATS_API_KEY"), values["SSTATS_API_KEY"]),
		BaseURL:   firstNonEmpty(os.Getenv("SSTATS_BASE_URL"), values["SSTATS_BASE_URL"]),
		RootDir:   rootDir,
		OutputPath: filepath.Join(rootDir, "collector", "leagues_raw.json"),
	}

	if strings.TrimSpace(config.APIKey) == "" {
		return Config{}, errors.New("SSTATS_API_KEY is required")
	}

	if strings.TrimSpace(config.BaseURL) == "" {
		return Config{}, errors.New("SSTATS_BASE_URL is required")
	}

	return config, nil
}

func findProjectRoot() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	current := workingDir
	for {
		envPath := filepath.Join(current, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("project root with .env was not found")
		}

		current = parent
	}
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env line: %s", line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}

	return values, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}