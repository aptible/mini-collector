package emitter

import (
	"context"
	"github.com/aptible/mega-collector/batch"
)

type Emitter interface {
	Emit(ctx context.Context, batch batch.Batch) error
	Close()
}
