/*

This is a fake emitter that notifies PagerDuty when events cannot be delivered
to other emitters. It's meant to be stacked on top of other Emitters.

*/
package notify

import (
	"context"
	"fmt"
	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/aptible/mega-collector/batch"
	"github.com/aptible/mega-collector/emitter"
	log "github.com/sirupsen/logrus"
)

type notifyEmitter struct {
	event *pagerduty.Event
}

func Open(config *Config) emitter.Emitter {
	description := fmt.Sprintf("Aggregator failed to Emit: %s", config.Identifier)

	event := pagerduty.Event{
		ServiceKey:  config.IntegrationKey,
		Type:        "trigger",
		IncidentKey: config.IncidentKey,
		Description: description,
	}

	return &notifyEmitter{event: &event}
}

func (e *notifyEmitter) Emit(ctx context.Context, batch batch.Batch) error {
	_, err := pagerduty.CreateEvent(*e.event)

	if err != nil {
		return fmt.Errorf("error notifying PagerDuty: %v", err)
	}

	log.WithFields(log.Fields{
		"source":  "emitter",
		"emitter": "notify",
	}).WithFields(
		batch.Fields(),
	).Errorf("Emit: notified (%s): %s", e.event.IncidentKey, e.event.Description)

	return nil
}

func (e *notifyEmitter) Close() {
}
