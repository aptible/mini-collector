package batcher

import (
	"context"
	"github.com/aptible/mega-collector/batch"
	"github.com/aptible/mega-collector/emitter"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	ingestBufferSize = 10
	emitTimeout      = time.Second
)

var logger = logrus.WithFields(logrus.Fields{
	"source": "batcher",
})

type batcher struct {
	emitter emitter.Emitter

	minPublishFrequency time.Duration
	maxBatchSize        int

	ingestBuffer chan *batch.Entry
	doneChannel  chan interface{}
	cancel       context.CancelFunc
}

func New(emitter emitter.Emitter, minPublishFrequency time.Duration, maxBatchSize int) Batcher {
	ctx, cancel := context.WithCancel(context.Background())

	b := &batcher{
		emitter:             emitter,
		minPublishFrequency: minPublishFrequency,
		maxBatchSize:        maxBatchSize,
		ingestBuffer:        make(chan *batch.Entry, ingestBufferSize),
		doneChannel:         make(chan interface{}),
		cancel:              cancel,
	}

	go func() {
		b.run(ctx)
		b.doneChannel <- nil
	}()

	return b
}

func (b *batcher) run(ctx context.Context) {
	var batchId uint64 = 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// no-op: proceed
		}

		nextBatch := func() batch.Batch {
			batchCtx, cancel := context.WithTimeout(ctx, b.minPublishFrequency)
			defer cancel()
			return b.prepareBatch(batchId, batchCtx)
		}()

		b.emitBatch(nextBatch)

		batchId++
	}

	// TODO: Need to drainBatch without the cancelled context here!
}
func (b *batcher) prepareBatch(id uint64, ctx context.Context) batch.Batch {
	currentBatch := batch.Batch{
		Id:      id,
		Entries: make([]batch.Entry, 0, b.maxBatchSize),
	}

	for {
		select {
		case newEntry := <-b.ingestBuffer:
			currentBatch.Entries = append(currentBatch.Entries, *newEntry)
			if len(currentBatch.Entries) >= b.maxBatchSize {
				return currentBatch
			}
		case <-ctx.Done():
			return currentBatch
		}
	}
}

func (b *batcher) emitBatch(batch batch.Batch) {
	logger.WithFields(
		batch.Fields(),
	).Infof("Emit: %d entries", len(batch.Entries))

	ctx, cancel := context.WithTimeout(context.Background(), emitTimeout)
	defer cancel()

	err := b.emitter.Emit(ctx, batch)
	if err != nil {
		logger.WithFields(
			batch.Fields(),
		).Errorf("Emit failed: %v", err)
	}
}

func (b *batcher) Ingest(ctx context.Context, entry *batch.Entry) error {
	// TODO: Same as everywhere, we need to stop Ingesting when we are
	// Closed!
	select {
	case b.ingestBuffer <- entry:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *batcher) Close() {
	logger.Info("shutting down")
	b.cancel()
	<-b.doneChannel
	logger.Info("shut down")
}
