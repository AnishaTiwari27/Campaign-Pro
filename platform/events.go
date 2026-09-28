package platform

import (
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
)

// EventBus is a thin wrapper around a NATS connection — just enough
// pub/sub for campaigns-service to announce `campaign.created` and for
// notifications-service / analytics-service's cache invalidation to react,
// without either side depending on the other directly.
type EventBus struct {
	nc *nats.Conn
}

func ConnectEventBus(url string) (*EventBus, error) {
	nc, err := nats.Connect(url, nats.Name("campaign-tracker-pro"))
	if err != nil {
		return nil, err
	}
	return &EventBus{nc: nc}, nil
}

func (b *EventBus) Close() {
	if b.nc != nil {
		b.nc.Close()
	}
}

// Publish JSON-encodes v and publishes it on subject. Errors are logged,
// not returned — a dropped notification/cache-invalidation event shouldn't
// fail the request that triggered it (the write to the DB already
// succeeded by the time this is called).
func (b *EventBus) Publish(subject string, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		slog.Error("event marshal", "subject", subject, "err", err)
		return
	}
	if err := b.nc.Publish(subject, payload); err != nil {
		slog.Error("event publish", "subject", subject, "err", err)
	}
}

// Subscribe registers handler for every message on subject. handler
// receives the raw JSON payload bytes so each subscriber decodes into
// whatever shape it needs.
func (b *EventBus) Subscribe(subject string, handler func([]byte)) error {
	_, err := b.nc.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	return err
}
