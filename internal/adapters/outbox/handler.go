package outbox

import (
	"context"
	"time"

	"github.com/rafaelleal24/challenge/internal/adapters/config"
	"github.com/rafaelleal24/challenge/internal/core/logger"
	"github.com/rafaelleal24/challenge/internal/core/port"
)

type Handler struct {
	outbox   Repository
	broker   port.BrokerPort
	interval time.Duration
	batch    int
}

func NewHandler(outbox Repository, broker port.BrokerPort, config config.OutboxConfig) *Handler {
	return &Handler{
		outbox:   outbox,
		broker:   broker,
		interval: config.Interval,
		batch:    config.BatchSize,
	}
}

func (h *Handler) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.processEvents(ctx)
		}
	}
}

func (h *Handler) processEvents(ctx context.Context) {
	entries, err := h.outbox.FetchPending(ctx, h.batch)
	if err != nil {
		logger.Error(ctx, "outbox: failed to fetch pending events", err, map[string]any{
			"batch": h.batch,
		})
		return
	}

	for _, entry := range entries {
		eventLogAttributes := map[string]any{
			"event_id":    entry.ID,
			"event_name":  entry.EventName,
			"entity_name": entry.EntityName,
		}
		if err := h.broker.PublishRaw(ctx, entry.EventName, entry.EntityName, entry.EventData); err != nil {
			logger.Error(ctx, "outbox: failed to publish event", err, eventLogAttributes)
			continue
		}

		logger.Debug(ctx, "outbox: event published", eventLogAttributes)

		if err := h.outbox.Delete(ctx, entry.ID); err != nil {
			logger.Error(ctx, "outbox: failed to delete event after publish", err, eventLogAttributes)
		}
	}
}
