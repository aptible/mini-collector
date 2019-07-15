package writer

import (
	"context"
	"fmt"
	"github.com/aptible/mega-collector/batch"
	"github.com/aptible/mega-collector/emitter"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	bufferSize   = 10
	drainTimeout = 2 * time.Second
)

type writerEmitter struct {
	logger *logrus.Entry

	writer      Writer
	nextEmitter emitter.Emitter

	sendBuffer chan batch.Batch

	doneChannel chan interface{}
	cancel      context.CancelFunc
}

// TODO: Update all!
func Open(name string, writer Writer, nextEmitter emitter.Emitter) emitter.Emitter {
	ctx, cancel := context.WithCancel(context.Background())

	em := &writerEmitter{
		logger: logrus.WithFields(logrus.Fields{
			"source":  "emitter",
			"emitter": name,
		}),

		writer:     writer,
		sendBuffer: make(chan batch.Batch, bufferSize),

		nextEmitter: nextEmitter,
		doneChannel: make(chan interface{}),
		cancel:      cancel,
	}

	go em.run(ctx)

	return em
}

func (e *writerEmitter) Emit(ctx context.Context, batch batch.Batch) error {
	if len(batch.Entries) <= 0 {
		// Nothing to emit, skip it.
		e.logger.WithFields(
			batch.Fields(),
		).Debugf("skipped")
		return nil
	}

	select {
	case e.sendBuffer <- batch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *writerEmitter) run(ctx context.Context) {
	e.logger.Infof("starting")

	for {
		// TODO: is this really the right ctx to use here? Presumably
		// we should layer on some emit timeout?
		if e.runOnce(ctx) {
			break
		}
	}

	e.logger.Infof("draining send buffer")

	// TODO: Should prevent further Emits at this stage.
	func() {
		drainCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
		defer cancel()

		e.drainSendBuffer(drainCtx)
	}()

	e.doneChannel <- nil
}

func (e *writerEmitter) runOnce(ctx context.Context) bool {
	select {
	case batch := <-e.sendBuffer:
		e.sendOrDelegateToNextEmitter(ctx, batch)
		return false
	case <-ctx.Done():
		return true
	}
}

func (e *writerEmitter) drainSendBuffer(ctx context.Context) error {
	for {
		select {
		case batch := <-e.sendBuffer:
			e.sendOrDelegateToNextEmitter(ctx, batch)
		default:
			return nil
		}
	}
}

func (e *writerEmitter) sendOrDelegateToNextEmitter(ctx context.Context, batch batch.Batch) {
	err := e.sendBatch(batch)

	if err != nil {
		e.logger.WithFields(
			batch.Fields(),
		).Warnf("sendBatch failed: %v", err)

		err = e.nextEmitter.Emit(ctx, batch)
		if err != nil {
			e.logger.WithFields(
				batch.Fields(),
			).Errorf("nextEmitter.Emit failed: %v", err)
		}
	}
}

func (e *writerEmitter) sendBatch(batch batch.Batch) error {
	e.logger.WithFields(
		batch.Fields(),
	).Debugf("sendBatch")

	err := e.writer.Write(batch)

	if err != nil {
		return fmt.Errorf("Write failed: %v", err)
	}

	e.logger.WithFields(
		batch.Fields(),
	).Infof("Write succeeded")

	return nil
}

func (e *writerEmitter) Close() {
	e.logger.Info("shutting down")
	e.cancel()
	<-e.doneChannel
	e.logger.Info("shut down")
}
