package batcher

import (
	"context"
	"github.com/aptible/mega-collector/batch"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type chanEmitter struct {
	batches chan batch.Batch
}

func (e *chanEmitter) Emit(ctx context.Context, batch batch.Batch) error {
	select {
	case e.batches <- batch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *chanEmitter) Close() {
}

func closeEverything(batcher Batcher, em *chanEmitter) {
	// The batcher is going to continue to want sending at its minimum
	// frequency even if we stopped reading from the chanEmitter so we have
	// to drain the emitter while we close it.
	go func() {
		for b := range em.batches {
			if len(b.Entries) == 0 {
				break
			}
		}
	}()

	batcher.Close()
	close(em.batches)
}

func TestBatcherCloses(t *testing.T) {
	em := &chanEmitter{batches: make(chan batch.Batch, 1)}
	b := New(em, time.Hour, 10)
	b.Close()
}

func TestBatcherEmitsAtMinimumFrequency(t *testing.T) {
	em := &chanEmitter{batches: make(chan batch.Batch)}
	b := New(em, time.Millisecond, 10)
	defer closeEverything(b, em)

	t0 := time.Now()
	i := 0

	for b := range em.batches {
		i++
		assert.Equal(t, 0, len(b.Entries))
		if i >= 10 {
			break
		}
	}

	t1 := time.Now()

	// NOTE: this accounts for some overhead
	assert.True(t, t1.Sub(t0) < 20*time.Millisecond)
}

func TestBatcherEmitsAtMaxBatchSize(t *testing.T) {
	em := &chanEmitter{batches: make(chan batch.Batch)}
	b := New(em, time.Hour, 10)
	defer closeEverything(b, em)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	for i := 0; i < 10; i++ {
		b.Ingest(ctx, &batch.Entry{Time: time.Unix(int64(i), 0)})
	}

	select {
	case batch := <-em.batches:
		assert.Equal(t, 10, len(batch.Entries))
		for i := 0; i < 10; i++ {
			assert.Equal(t, int64(i), batch.Entries[i].Time.Unix())
		}
	case err := <-ctx.Done():
		t.Fatalf("batch was not delivered: %v", err)
	}
}
