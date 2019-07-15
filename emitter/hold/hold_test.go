package hold

import (
	"context"
	"github.com/aptible/mega-collector/batch"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testEmitter struct {
	batches chan (batch.Batch)
}

func (em *testEmitter) Emit(ctx context.Context, batch batch.Batch) error {
	select {
	case em.batches <- batch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (em *testEmitter) Close() {
	return
}

func TestItHoldsThenReleasesTheBatch(t *testing.T) {
	next := &testEmitter{batches: make(chan batch.Batch, 1)}
	em := Open(4*time.Millisecond, next)
	defer em.Close()

	sent := batch.Batch{Id: 123}
	err := em.Emit(context.Background(), sent)

	if assert.Nil(t, err) {
		select {
		case <-next.batches:
			t.Fatal("got it too fast")
		case <-time.After(2 * time.Millisecond):
		}

		select {
		case got := <-next.batches:
			assert.Equal(t, sent.Id, got.Id)
		case <-time.After(4 * time.Millisecond):
			t.Fatal("got it too slow")
		}
	}
}

func TestItReleasesAllBatchesWhenClosing(t *testing.T) {
	next := &testEmitter{batches: make(chan batch.Batch, 1)}
	em := Open(4*time.Millisecond, next)

	err := em.Emit(context.Background(), batch.Batch{})

	if assert.Nil(t, err) {
		em.Close()

		select {
		case <-next.batches:
		case <-time.After(1 * time.Millisecond):
			t.Fatal("got it too slow")
		}
	}
}

func TestItEventuallyGivesUp(t *testing.T) {
	next := &testEmitter{batches: make(chan batch.Batch)}
	em := Open(4*time.Millisecond, next).(*holdEmitter)
	em.delegateTimeout = time.Millisecond

	err := em.Emit(context.Background(), batch.Batch{})

	if assert.Nil(t, err) {
		em.Close()

		select {
		case <-next.batches:
			t.Fatal("unexpectedly got it")
		default:
		}
	}
}
