package batch

import (
	"github.com/aptible/mega-collector/api"
	"time"
)

type Entry struct {
	Time time.Time
	Tags map[string]string
	api.PublishRequest
}
