package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONLinesWriter struct {
	file *os.File
	mu   sync.Mutex
	enc  *json.Encoder
}

func NewJSONLinesWriter(path string) (*JSONLinesWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create output file: %w", err)
	}

	return &JSONLinesWriter{
		file: file,
		enc:  json.NewEncoder(file),
	}, nil
	}

func (writer *JSONLinesWriter) Write(value any) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if err := writer.enc.Encode(value); err != nil {
		return fmt.Errorf("encode json line: %w", err)
	}

	return nil
}

func (writer *JSONLinesWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.file == nil {
		return nil
	}

	err := writer.file.Close()
	writer.file = nil
	return err
}