package batcher

import (
	"context"
	"github.com/aptible/mega-collector/batch"
)

type Batcher interface {
	Ingest(ctx context.Context, entry *batch.Entry) error
	Close()
}
