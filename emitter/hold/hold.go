package hold

import (
	"context"
	"fmt"
	"github.com/aptible/mega-collector/batch"
	"github.com/aptible/mega-collector/emitter"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	defaultDelegateTimeout = 2 * time.Second
)

type holdEmitter struct {
	logger *logrus.Entry

	nextEmitter emitter.Emitter
	delay       time.Duration

	delegateTimeout time.Duration

	context context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func Open(delay time.Duration, nextEmitter emitter.Emitter) emitter.Emitter {
	ctx, cancel := context.WithCancel(context.Background())

	return &holdEmitter{
		logger: logrus.WithFields(logrus.Fields{
			"source":  "emitter",
			"emitter": fmt.Sprintf("hold for %s", delay),
		}),

		nextEmitter: nextEmitter,
		delay:       delay,

		delegateTimeout: defaultDelegateTimeout,

		context: ctx,
		cancel:  cancel,
		wg:      sync.WaitGroup{},
	}
}

func (em *holdEmitter) Emit(ctx context.Context, batch batch.Batch) error {
	em.wg.Add(1)

	// We ignore the incoming context: we're going to kick off a new
	// goroutine immediatley and accept the message (technically we should
	// wait on the lock for that)
	go func() {
		defer em.wg.Done()
		em.holdThenDelegateToNextEmitter(batch)
	}()

	return nil
}

func (em *holdEmitter) Close() {
	// TODO: This will race with Emit. More broadly speaking, you can call
	// Close() *then* Emit().
	em.logger.Info("shutting down")
	em.cancel()
	em.wg.Wait()
	em.logger.Info("shut down")
}

func (em *holdEmitter) holdThenDelegateToNextEmitter(batch batch.Batch) {
	select {
	case <-em.context.Done():
		// We're closing, stop holding!
	case <-time.After(em.delay):
		// We're done waiting for this one.:w
	}

	ctx, cancel := context.WithTimeout(context.Background(), em.delegateTimeout)
	defer cancel()

	err := em.nextEmitter.Emit(ctx, batch)

	if err != nil {
		em.logger.WithFields(
			batch.Fields(),
		).Errorf("nextEmitter.Emit failed: %v", err)
	}
}
