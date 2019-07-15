package writer

import (
	"github.com/aptible/mega-collector/batch"
)

type Writer interface {
	Write(batch batch.Batch) error
}

type CloseWriter interface {
	Writer
	Close() error
}
