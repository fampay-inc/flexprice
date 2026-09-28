package publisher

import (
	"context"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/flexprice/flexprice/internal/config"
	"github.com/flexprice/flexprice/internal/logger"
	"github.com/flexprice/flexprice/internal/pubsub"
	repoent "github.com/flexprice/flexprice/internal/repository/ent"
	"github.com/flexprice/flexprice/internal/tracing"
	"github.com/flexprice/flexprice/internal/types"
)

// MessagePublisher publishes messages to a topic (e.g. Kafka producer).
// Used when webhook delivery is Kafka-backed so the publisher can use the shared producer.
// Signature matches watermill message.Publisher (variadic Publish).
type MessagePublisher interface {
	Publish(topic string, msg ...*message.Message) error
}

// WebhookPublisher interface for producing webhook events
type WebhookPublisher interface {
	PublishWebhook(ctx context.Context, event *types.WebhookEvent) error
	Close() error
}

// webhookPublisher publishes webhook events to a topic (memory PubSub or shared Kafka producer).
type webhookPublisher struct {
	pubSub          pubsub.PubSub    // used when pubsub is memory
	producer        MessagePublisher // used when pubsub is kafka (shared producer); Close is no-op
	config          *config.Webhook
	logger          *logger.Logger
	systemEventRepo *repoent.SystemEventRepository
	tracing         *tracing.Service
}

// NewPublisher creates a webhook publisher backed by a PubSub (e.g. in-memory for tests/local).
func NewPublisher(
	pubSub pubsub.PubSub,
	cfg *config.Configuration,
	logger *logger.Logger,
	systemEventRepo *repoent.SystemEventRepository,
	tracingSvc *tracing.Service,
) (WebhookPublisher, error) {
	return &webhookPublisher{
		pubSub:          pubSub,
		config:          &cfg.Webhook,
		logger:          logger,
		systemEventRepo: systemEventRepo,
		tracing:         tracingSvc,
	}, nil
}

// NewPublisherFromProducer creates a webhook publisher backed by a shared message.Publisher (e.g. Kafka producer).
// Close() is a no-op; the shared producer is closed by fx lifecycle.
func NewPublisherFromProducer(
	producer MessagePublisher,
	cfg *config.Configuration,
	logger *logger.Logger,
	systemEventRepo *repoent.SystemEventRepository,
	tracingSvc *tracing.Service,
) (WebhookPublisher, error) {
	return &webhookPublisher{
		producer:        producer,
		config:          &cfg.Webhook,
		logger:          logger,
		systemEventRepo: systemEventRepo,
		tracing:         tracingSvc,
	}, nil
}

func (p *webhookPublisher) PublishWebhook(ctx context.Context, event *types.WebhookEvent) (err error) {
	transport := "pubsub"
	if p.producer != nil {
		transport = "kafka"
	}

	rootSpan, ctx := p.tracing.StartWebhookPublishSpan(ctx, map[string]interface{}{
		"event.id":       event.ID,
		"event.name":     string(event.EventName),
		"entity.type":    string(event.EntityType),
		"entity.id":      event.EntityID,
		"tenant.id":      event.TenantID,
		"environment.id": event.EnvironmentID,
		"topic":          p.config.Topic,
		"transport":      transport,
	})
	defer func() {
		if rootSpan == nil {
			return
		}
		if err != nil {
			rootSpan.SetStatusError(err)
		} else {
			rootSpan.SetStatusOK()
		}
		rootSpan.Finish()
	}()

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if rootSpan != nil {
		rootSpan.SetData("payload.bytes", len(payload))
	}

	messageID := event.ID
	if messageID == "" {
		messageID = watermill.NewUUID()
	}

	msg := message.NewMessage(messageID, payload)
	msg.Metadata.Set("tenant_id", event.TenantID)
	msg.Metadata.Set("environment_id", event.EnvironmentID)
	msg.Metadata.Set("user_id", event.UserID)

	p.logger.Info(ctx, "publishing webhook event",
		"event_id", event.ID,
		"event_name", event.EventName,
		"entity_id", event.EntityID,
		"tenant_id", event.TenantID,
		"topic", p.config.Topic,
		"payload", string(payload),
	)

	if p.systemEventRepo != nil {
		dbSpan, dbCtx := p.tracing.StartDBSpan(ctx, "system_events.on_consumed", map[string]interface{}{
			"event.id":    event.ID,
			"event.name":  string(event.EventName),
			"entity.type": string(event.EntityType),
			"entity.id":   event.EntityID,
		})
		onConsumedErr := p.systemEventRepo.OnConsumed(dbCtx, event)
		if dbSpan != nil {
			if onConsumedErr != nil {
				dbSpan.SetStatusError(onConsumedErr)
			} else {
				dbSpan.SetStatusOK()
			}
			dbSpan.Finish()
		}
		if onConsumedErr != nil {
			p.logger.Error(ctx, "system_events OnConsumed failed",
				"error", onConsumedErr,
				"event_id", event.ID,
				"event_name", event.EventName,
			)
		}
	}

	produceSpan, produceCtx := p.tracing.StartKafkaProducerSpan(ctx, p.config.Topic, map[string]interface{}{
		"event.id":       event.ID,
		"event.name":     string(event.EventName),
		"tenant.id":      event.TenantID,
		"environment.id": event.EnvironmentID,
		"message.id":     messageID,
		"payload.bytes":  len(payload),
		"transport":      transport,
	})

	if p.producer != nil {
		err = p.producer.Publish(p.config.Topic, msg)
	} else {
		err = p.pubSub.Publish(produceCtx, p.config.Topic, msg)
	}

	if produceSpan != nil {
		if err != nil {
			produceSpan.SetStatusError(err)
		} else {
			produceSpan.SetStatusOK()
		}
		produceSpan.Finish()
	}

	if err != nil {
		p.logger.Error(ctx, "failed to publish webhook event",
			"error", err,
			"event_id", event.ID,
			"event_name", event.EventName,
			"tenant_id", event.TenantID,
		)
		return err
	}

	p.logger.Debug(ctx, "successfully published webhook event",
		"event_id", event.ID,
		"event_name", event.EventName,
		"tenant_id", event.TenantID,
	)

	return nil
}

// Close closes the publisher. No-op when using shared Kafka producer (lifecycle-managed).
func (p *webhookPublisher) Close() error {
	if p.producer != nil {
		return nil
	}
	return p.pubSub.Close()
}
