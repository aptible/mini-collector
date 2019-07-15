package blackhole

import (
	"context"
	"github.com/aptible/mega-collector/batch"
	"github.com/aptible/mega-collector/emitter"
	log "github.com/sirupsen/logrus"
)

type blackholeEmitter struct{}

func Open() emitter.Emitter {
	return &blackholeEmitter{}
}

func (e *blackholeEmitter) Emit(ctx context.Context, batch batch.Batch) error {
	log.WithFields(log.Fields{
		"source":  "emitter",
		"emitter": "blackhole",
	}).WithFields(
		batch.Fields(),
	).Errorf("batch lost")

	return nil
}

func (e *blackholeEmitter) Close() {
}
