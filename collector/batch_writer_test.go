package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockWriter struct {
	writes     []any
	flushCount int
	closeCount int
	writeErr   error
	flushErr   error
}

func (m *mockWriter) Write(value any) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writes = append(m.writes, value)
	return nil
}

func (m *mockWriter) Flush() error {
	m.flushCount++
	return m.flushErr
}

func (m *mockWriter) Close() error {
	m.closeCount++
	return nil
}

func TestBatchWriter_FlushByBatchSize(t *testing.T) {
	inner := &mockWriter{}
	bw := NewBatchWriter(context.Background(), inner, 3, 10*time.Second)
	bw.Start()

	for i := 1; i <= 3; i++ {
		if err := bw.Write(League{ID: int64(i)}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	if len(inner.writes) != 3 {
		t.Errorf("expected 3 writes, got %d", len(inner.writes))
	}
	if inner.flushCount != 1 {
		t.Errorf("expected 1 flush, got %d", inner.flushCount)
	}

	bw.Close()
}

func TestBatchWriter_FlushByTimer(t *testing.T) {
	inner := &mockWriter{}
	bw := NewBatchWriter(context.Background(), inner, 50, 200*time.Millisecond)
	bw.Start()

	if err := bw.Write(League{ID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(inner.writes) != 0 {
		t.Errorf("expected 0 writes before timer, got %d", len(inner.writes))
	}

	time.Sleep(300 * time.Millisecond)

	if len(inner.writes) != 1 {
		t.Errorf("expected 1 write after timer, got %d", len(inner.writes))
	}
	if inner.flushCount != 1 {
		t.Errorf("expected 1 flush after timer, got %d", inner.flushCount)
	}

	bw.Close()
}

func TestBatchWriter_FlushOnClose(t *testing.T) {
	inner := &mockWriter{}
	bw := NewBatchWriter(context.Background(), inner, 50, 10*time.Second)
	bw.Start()

	if err := bw.Write(League{ID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bw.Close()

	if len(inner.writes) != 1 {
		t.Errorf("expected 1 write after close, got %d", len(inner.writes))
	}
	if inner.flushCount < 1 {
		t.Errorf("expected at least 1 flush after close, got %d", inner.flushCount)
	}
}

func TestBatchWriter_EmptyClose(t *testing.T) {
	inner := &mockWriter{}
	bw := NewBatchWriter(context.Background(), inner, 50, 10*time.Second)
	bw.Start()

	bw.Close()

	if len(inner.writes) != 0 {
		t.Errorf("expected 0 writes, got %d", len(inner.writes))
	}
}

func TestBatchWriter_MultipleBatches(t *testing.T) {
	inner := &mockWriter{}
	bw := NewBatchWriter(context.Background(), inner, 2, 10*time.Second)
	bw.Start()

	for i := 1; i <= 5; i++ {
		if err := bw.Write(League{ID: int64(i)}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	if len(inner.writes) != 4 {
		t.Errorf("expected 4 writes (2 batches), got %d", len(inner.writes))
	}
	if inner.flushCount != 2 {
		t.Errorf("expected 2 flushes, got %d", inner.flushCount)
	}

	bw.Close()

	if len(inner.writes) != 5 {
		t.Errorf("expected 5 writes total, got %d", len(inner.writes))
	}
}

func TestBatchWriter_WriteError(t *testing.T) {
	inner := &mockWriter{writeErr: errors.New("write failed")}
	bw := NewBatchWriter(context.Background(), inner, 2, 10*time.Second)
	bw.Start()

	if err := bw.Write(League{ID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bw.Write(League{ID: 2}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	closeErr := bw.Close()
	if closeErr == nil {
		t.Fatal("expected error on close due to write failure")
	}
}

func TestBatchWriter_FlushError(t *testing.T) {
	inner := &mockWriter{flushErr: errors.New("flush failed")}
	bw := NewBatchWriter(context.Background(), inner, 2, 10*time.Second)
	bw.Start()

	if err := bw.Write(League{ID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bw.Write(League{ID: 2}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	closeErr := bw.Close()
	if closeErr == nil {
		t.Fatal("expected error on close due to flush failure")
	}
}

func TestBatchWriter_WriteAfterClose(t *testing.T) {
	inner := &mockWriter{}
	bw := NewBatchWriter(context.Background(), inner, 50, 10*time.Second)
	bw.Start()
	bw.Close()

	err := bw.Write(League{ID: 1})
	if err == nil {
		t.Fatal("expected error when writing after close")
	}
}

func TestBatchWriter_TimerResetsAfterBatchFlush(t *testing.T) {
	inner := &mockWriter{}
	bw := NewBatchWriter(context.Background(), inner, 2, 200*time.Millisecond)
	bw.Start()

	bw.Write(League{ID: 1})
	bw.Write(League{ID: 2})

	time.Sleep(100 * time.Millisecond)

	if inner.flushCount != 1 {
		t.Fatalf("expected 1 flush, got %d", inner.flushCount)
	}

	bw.Write(League{ID: 3})

	time.Sleep(300 * time.Millisecond)

	if inner.flushCount != 2 {
		t.Errorf("expected 2 flushes (timer should reset), got %d", inner.flushCount)
	}

	bw.Close()
}

func TestBatchWriter_ContextCancellation(t *testing.T) {
	inner := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())
	bw := NewBatchWriter(ctx, inner, 50, 10*time.Second)
	bw.Start()

	if err := bw.Write(League{ID: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancel()
	time.Sleep(100 * time.Millisecond)

	err := bw.Write(League{ID: 2})
	if err == nil {
		t.Fatal("expected error after context cancellation")
	}

	bw.Close()
}

func TestBatchWriter_GracefulDrain(t *testing.T) {
	inner := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())
	bw := NewBatchWriter(ctx, inner, 50, 10*time.Second)
	bw.Start()

	for i := 1; i <= 10; i++ {
		if err := bw.Write(League{ID: int64(i)}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	bw.Close()

	if len(inner.writes) != 10 {
		t.Errorf("expected 10 writes after graceful drain, got %d", len(inner.writes))
	}
	if inner.flushCount < 1 {
		t.Errorf("expected at least 1 flush, got %d", inner.flushCount)
	}
}

func TestBatchWriter_GracefulDrainPartialBatch(t *testing.T) {
	inner := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())
	bw := NewBatchWriter(ctx, inner, 10, 10*time.Second)
	bw.Start()

	for i := 1; i <= 3; i++ {
		if err := bw.Write(League{ID: int64(i)}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	cancel()
	bw.Close()

	if len(inner.writes) != 3 {
		t.Errorf("expected 3 writes, got %d", len(inner.writes))
	}
	if inner.flushCount < 1 {
		t.Errorf("expected at least 1 flush, got %d", inner.flushCount)
	}
}

func TestBatchWriter_GracefulDrainMultipleBatches(t *testing.T) {
	inner := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())
	bw := NewBatchWriter(ctx, inner, 3, 10*time.Second)
	bw.Start()

	for i := 1; i <= 7; i++ {
		if err := bw.Write(League{ID: int64(i)}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	cancel()
	bw.Close()

	if len(inner.writes) != 7 {
		t.Errorf("expected 7 writes, got %d", len(inner.writes))
	}
	if inner.flushCount != 3 {
		t.Errorf("expected 3 flushes (2 full + 1 partial), got %d", inner.flushCount)
	}
}

func TestBatchWriter_GracefulDrainEmptyChannel(t *testing.T) {
	inner := &mockWriter{}
	ctx, cancel := context.WithCancel(context.Background())
	bw := NewBatchWriter(ctx, inner, 10, 10*time.Second)
	bw.Start()

	cancel()
	bw.Close()

	if len(inner.writes) != 0 {
		t.Errorf("expected 0 writes, got %d", len(inner.writes))
	}
	if inner.flushCount != 0 {
		t.Errorf("expected 0 flushes, got %d", inner.flushCount)
	}
}
