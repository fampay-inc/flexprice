package service

import (
	"context"
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/flexprice/flexprice/internal/config"
	domainBenefit "github.com/flexprice/flexprice/internal/domain/benefit"
	ierr "github.com/flexprice/flexprice/internal/errors"
	"github.com/flexprice/flexprice/internal/pubsub"
	"github.com/flexprice/flexprice/internal/pubsub/kafka"
	pubsubRouter "github.com/flexprice/flexprice/internal/pubsub/router"
	"github.com/flexprice/flexprice/internal/sentry"
	"github.com/flexprice/flexprice/internal/types"
	"github.com/google/uuid"
	benefitsv1 "gitlab.famapp.in/backend/flexprice/protos/pb/v1"
	"google.golang.org/protobuf/proto"
)

type dropEvent struct {
	reason string
}

func (e *dropEvent) Error() string { return e.reason }

func drop(reason string) *dropEvent { return &dropEvent{reason: reason} }

func isDropEvent(err error) bool {
	var de *dropEvent
	return errors.As(err, &de)
}

type eventValidation struct {
	Product    string
	CustomerID string
}

type BenefitConsumptionService interface {
	RegisterHandler(router *pubsubRouter.Router, cfg *config.Configuration)
}

type benefitConsumptionService struct {
	ServiceParams
	pubSub        pubsub.PubSub
	sentryService *sentry.Service
}

func NewBenefitConsumptionService(
	params ServiceParams,
	sentryService *sentry.Service,
) BenefitConsumptionService {
	s := &benefitConsumptionService{
		ServiceParams: params,
		sentryService: sentryService,
	}

	ps, err := kafka.NewPubSubFromConfig(
		params.Config,
		params.Logger,
		params.Config.BenefitEvents.ConsumerGroup,
	)
	if err != nil {
		params.Logger.Fatalw("failed to create benefit events pubsub", "error", err)
		return nil
	}
	s.pubSub = ps
	return s
}

func (s *benefitConsumptionService) RegisterHandler(router *pubsubRouter.Router, cfg *config.Configuration) {
	if !cfg.BenefitEvents.Enabled {
		s.Logger.Infow("benefit consumption handler disabled by configuration")
		return
	}

	throttle := middleware.NewThrottle(cfg.BenefitEvents.RateLimit, time.Second)

	router.AddNoPublishHandler(
		"benefit_consumption_handler",
		cfg.BenefitEvents.Topic,
		s.pubSub,
		s.processMessage,
		throttle.Middleware,
	)

	s.Logger.Infow("registered benefit consumption handler",
		"topic", cfg.BenefitEvents.Topic,
		"rate_limit", cfg.BenefitEvents.RateLimit,
	)
}

func (s *benefitConsumptionService) processMessage(msg *message.Message) error {
	var ev benefitsv1.BenefitEvent
	if err := proto.Unmarshal(msg.Payload, &ev); err != nil {
		s.Logger.Errorw("failed to unmarshal benefit event proto",
			"error", err,
			"payload_len", len(msg.Payload),
		)
		s.sentryService.CaptureException(err)
		return nil
	}

	if err := validateProtoFields(&ev); err != nil {
		s.Logger.Warnw("dropping invalid benefit event",
			"reason", err.Error(),
			"event_id", ev.GetEventId(),
		)
		return nil
	}

	tenantID := s.Config.Billing.TenantID
	if tenantID == "" {
		s.Logger.Errorw("billing.tenant_id is not configured; cannot process benefit events",
			"event_id", ev.GetEventId())
		return nil
	}

	ctx := context.WithValue(context.Background(), types.CtxTenantID, tenantID)
	if environmentID := s.Config.Billing.EnvironmentID; environmentID != "" {
		ctx = context.WithValue(ctx, types.CtxEnvironmentID, environmentID)
	}

	validated, err := s.validateEvent(ctx, &ev)
	if err != nil {
		if isDropEvent(err) {
			s.Logger.Warnw("dropping invalid benefit event",
				"reason", err.Error(),
				"event_id", ev.GetEventId(),
				"subscription_id", ev.GetSubscriptionId(),
			)
			return nil
		}
		s.Logger.Errorw("benefit event validation errored, will retry",
			"error", err,
			"event_id", ev.GetEventId(),
		)
		return ierr.WithError(err).
			WithHint("Failed to validate benefit event").
			Mark(ierr.ErrSystem)
	}

	row := toLedgerRow(&ev, validated, tenantID, s.Config.Billing.EnvironmentID)

	if err := s.BenefitLedgerRepo.Create(ctx, row); err != nil {
		if ierr.IsAlreadyExists(err) {
			s.Logger.Debugw("duplicate benefit event ignored", "event_id", ev.GetEventId())
			return nil
		}
		s.Logger.Errorw("failed to store benefit event",
			"error", err,
			"event_id", ev.GetEventId(),
		)
		if !s.shouldRetryError(err) {
			return nil
		}
		return ierr.WithError(err).
			WithHint("Failed to store benefit event").
			Mark(ierr.ErrSystem)
	}

	s.Logger.Debugw("stored benefit event",
		"event_id", row.EventID,
	)
	return nil
}

func validateProtoFields(ev *benefitsv1.BenefitEvent) error {
	if ev.GetEventId() == "" || ev.GetSubscriptionId() == "" || ev.GetCycleId() == "" || ev.GetFeatureId() == "" {
		return drop("missing required fields or fields empty")
	}

	for name, val := range map[string]string{
		"subscription_id": ev.GetSubscriptionId(),
		"cycle_id":        ev.GetCycleId(),
		"feature_id":      ev.GetFeatureId(),
	} {
		if _, err := uuid.Parse(val); err != nil {
			return drop(name + " is not a valid UUID")
		}
	}
	return nil
}

func (s *benefitConsumptionService) validateEvent(ctx context.Context, ev *benefitsv1.BenefitEvent) (*eventValidation, error) {
	sub, err := s.SubRepo.Get(ctx, ev.GetSubscriptionId())
	if err != nil {
		if ierr.IsNotFound(err) {
			return nil, drop("subscription not found")
		}
		return nil, ierr.WithError(err).WithHint("subscription lookup failed").Mark(ierr.ErrDatabase)
	}

	if err := s.validateFeatureEntitlement(ctx, sub.PlanID, ev.GetFeatureId()); err != nil {
		return nil, err
	}

	inv, err := s.InvoiceRepo.Get(ctx, ev.GetCycleId())
	if err != nil {
		if ierr.IsNotFound(err) {
			return nil, drop("invoice (cycle_id) not found")
		}
		return nil, ierr.WithError(err).WithHint("invoice lookup failed").Mark(ierr.ErrDatabase)
	}
	if inv.SubscriptionID == nil || *inv.SubscriptionID != ev.GetSubscriptionId() {
		return nil, drop("invoice does not belong to subscription")
	}
	if inv.InvoiceStatus != types.InvoiceStatusFinalized {
		return nil, drop("invoice is not finalized")
	}
	if inv.PaymentStatus != types.PaymentStatusSucceeded {
		return nil, drop("invoice payment is not succeeded")
	}

	return &eventValidation{Product: *sub.Sku, CustomerID: sub.CustomerID}, nil
}

func (s *benefitConsumptionService) validateFeatureEntitlement(ctx context.Context, planID, featureID string) error {
	planEnts, err := s.EntitlementRepo.ListByPlanIDs(ctx, []string{planID})
	if err != nil {
		return ierr.WithError(err).WithHint("plan entitlement lookup failed").Mark(ierr.ErrDatabase)
	}
	for _, e := range planEnts {
		if e != nil && e.FeatureID == featureID && e.IsEnabled {
			return nil
		}
	}
	return drop("plan does not grant this feature")
}

func toLedgerRow(
	ev *benefitsv1.BenefitEvent,
	v *eventValidation,
	tenantID string,
	environmentID string,
) *domainBenefit.BenefitLedger {
	now := time.Now().UTC()
	row := &domainBenefit.BenefitLedger{
		ID:             types.GenerateUUID(),
		EventID:        ev.GetEventId(),
		SubscriptionID: ev.GetSubscriptionId(),
		CustomerID:     v.CustomerID,
		Product:        v.Product,
		CycleID:        ev.GetCycleId(),
		Category:       ev.GetCategory(),
		FeatureID:      ev.GetFeatureId(),
		Value:          int(ev.GetValue()),
		EventTimestamp: time.Unix(ev.GetTimestamp(), 0).UTC(),
		EnvironmentID:  environmentID,
	}
	row.TenantID = tenantID
	row.Status = types.StatusPublished
	row.CreatedAt = now
	row.UpdatedAt = now
	return row
}

func (s *benefitConsumptionService) shouldRetryError(err error) bool {
	if errors.Is(err, ierr.ErrValidation) ||
		errors.Is(err, ierr.ErrNotFound) ||
		errors.Is(err, ierr.ErrAlreadyExists) {
		return false
	}
	return true
}
