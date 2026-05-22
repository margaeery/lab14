package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONLinesWriter struct {
	file *os.File
	bw   *bufio.Writer
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

	bw := bufio.NewWriter(file)
	return &JSONLinesWriter{
		file: file,
		bw:   bw,
		enc:  json.NewEncoder(bw),
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

func (writer *JSONLinesWriter) Flush() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.bw == nil {
		return nil
	}

	if err := writer.bw.Flush(); err != nil {
		return fmt.Errorf("flush buffer: %w", err)
	}

	return nil
}

func (writer *JSONLinesWriter) Close() error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if writer.file == nil {
		return nil
	}

	_ = writer.bw.Flush()
	err := writer.file.Close()
	writer.file = nil
	return err
}