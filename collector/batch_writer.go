package main

import (
	"context"
	"sync"
	"time"
)

type BatchWriter struct {
	inner       Writer
	batchSize   int
	flushPeriod time.Duration
	channel     chan League
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	errCh       chan error
}

func NewBatchWriter(inner Writer, batchSize int, flushPeriod time.Duration) *BatchWriter {
	ctx, cancel := context.WithCancel(context.Background())
	return &BatchWriter{
		inner:       inner,
		batchSize:   batchSize,
		flushPeriod: flushPeriod,
		channel:     make(chan League, 100),
		ctx:         ctx,
		cancel:      cancel,
		errCh:       make(chan error, 1),
	}
}

func (bw *BatchWriter) Start() {
	bw.wg.Add(1)
	go bw.run()
}

func (bw *BatchWriter) Write(league League) error {
	select {
	case <-bw.ctx.Done():
		return bw.ctx.Err()
	case bw.channel <- league:
		return nil
	}
}

func (bw *BatchWriter) Close() error {
	bw.cancel()
	close(bw.channel)
	bw.wg.Wait()

	select {
	case err := <-bw.errCh:
		return err
	default:
		return nil
	}
}

func (bw *BatchWriter) run() {
	defer bw.wg.Done()

	batch := make([]League, 0, bw.batchSize)
	timer := time.NewTimer(bw.flushPeriod)
	defer timer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		for _, league := range batch {
			if err := bw.inner.Write(league); err != nil {
				select {
				case bw.errCh <- err:
				default:
				}
				return
			}
		}
		if err := bw.inner.Flush(); err != nil {
			select {
			case bw.errCh <- err:
				default:
				}
			return
		}
		batch = batch[:0]
	}

	for {
		select {
		case league, ok := <-bw.channel:
			if !ok {
				flush()
				return
			}
			batch = append(batch, league)
			if len(batch) >= bw.batchSize {
				flush()
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(bw.flushPeriod)
			}
		case <-timer.C:
			flush()
			timer.Reset(bw.flushPeriod)
		}
	}
}
