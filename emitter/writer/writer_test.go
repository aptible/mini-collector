package writer

import (
	"context"
	"fmt"
	"github.com/aptible/mega-collector/batch"
	"github.com/aptible/mega-collector/emitter/blackhole"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	writeTimeout = 5 * time.Millisecond
)

type errorWriter struct{}

func (c *errorWriter) Write(batch batch.Batch) error {
	return fmt.Errorf("permanent failure")
}

type successWriter struct {
	writes chan (batch.Batch)
}

func (c *successWriter) Write(batch batch.Batch) error {
	c.writes <- batch
	return nil
}

func TestEmitEmptyBatchDoesNotDoAnything(t *testing.T) {
	c := &successWriter{writes: make(chan batch.Batch)}
	em := Open("em", c, blackhole.Open())
	defer em.Close()

	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	err := em.Emit(ctx, batch.Batch{})
	if assert.Nil(t, err) {
		select {
		case <-c.writes:
			t.Fatalf("received write")
		case <-ctx.Done():
			// what we expected here
		}
	}
}

func TestEmitSendsToInfluxdb(t *testing.T) {
	c := &successWriter{writes: make(chan batch.Batch)}
	em := Open("em", c, blackhole.Open())
	defer em.Close()

	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	t0 := time.Unix(10, 0)

	err := em.Emit(ctx, batch.Batch{
		Entries: []batch.Entry{batch.Entry{Time: t0}},
	})

	if assert.Nil(t, err) {
		select {
		case b := <-c.writes:
			assert.Equal(t, 1, len(b.Entries))
			assert.Equal(t, t0, b.Entries[0].Time)
		case <-ctx.Done():
			t.Fatalf("did not receive write")
		}
	}
}

func TestEmitterPassesToNextEmitter(t *testing.T) {
	n := 2

	c0 := &successWriter{writes: make(chan batch.Batch)}
	em0 := Open("em0", c0, blackhole.Open())
	defer em0.Close()

	c1 := &errorWriter{}
	em1 := Open("em0", c1, em0)
	defer em1.Close()

	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	for i := 0; i < n; i++ {
		err := em1.Emit(ctx, batch.Batch{
			Entries: []batch.Entry{batch.Entry{}},
		})
		assert.Nil(t, err)
	}

	for i := 0; i < n; i++ {
		// Expect to have received n+1 writes.
		select {
		case <-c0.writes:
			// What we want
		case <-ctx.Done():
			t.Fatalf("did not receive write %d", i)
		}
	}
}
