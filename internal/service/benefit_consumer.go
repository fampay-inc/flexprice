package service

import (
	"context"
	"errors"
	"strings"
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
	benefitsv1 "gitlab.famapp.in/backend/flexprice/protos/pb/v1"
	"google.golang.org/protobuf/proto"
)

const categoryEnumPrefix = "BENEFIT_EVENT_CATEGORY_"

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
		return ierr.WithError(err).
			WithHint("invalid benefit event protobuf").
			Mark(ierr.ErrValidation)
	}

	if ev.GetEventId() == "" || ev.GetSubscriptionId() == "" ||
		ev.GetExternalCustomerId() == "" || ev.GetCycleId() == "" || ev.GetFeatureId() == "" {
		s.Logger.Warnw("dropping invalid benefit event: missing required fields",
			"event_id", ev.GetEventId(),
			"subscription_id", ev.GetSubscriptionId(),
			"cycle_id", ev.GetCycleId(),
			"feature_id", ev.GetFeatureId(),
		)
		return nil
	}
	tenantID := s.Config.Billing.TenantID
	environmentID := s.Config.Billing.EnvironmentID

	if tenantID == "" {
		s.Logger.Errorw("billing.tenant_id is not configured; cannot process benefit events",
			"event_id", ev.GetEventId())
		return ierr.NewError("billing tenant not configured").
			WithHint("Set billing.tenant_id in config").
			Mark(ierr.ErrSystem)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, types.CtxTenantID, tenantID)
	if environmentID != "" {
		ctx = context.WithValue(ctx, types.CtxEnvironmentID, environmentID)
	}

	sku, customerID, dropReason, retryErr := s.validateEvent(ctx, &ev)
	if retryErr != nil {
		s.Logger.Errorw("benefit event validation errored, will retry",
			"error", retryErr, "event_id", ev.GetEventId())
		return ierr.WithError(retryErr).
			WithHint("Failed to validate benefit event").
			Mark(ierr.ErrSystem)
	}
	if dropReason != "" {
		s.Logger.Warnw("dropping benefit event: "+dropReason,
			"event_id", ev.GetEventId(),
			"subscription_id", ev.GetSubscriptionId(),
			"external_customer_id", ev.GetExternalCustomerId(),
			"cycle_id", ev.GetCycleId(),
			"feature_id", ev.GetFeatureId(),
		)
		return nil
	}

	row := toLedgerRow(&ev, sku, customerID, tenantID, environmentID)

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
		"sku", row.SKU,
		"category", row.Category,
		"value", row.Value,
	)
	return nil
}

func toLedgerRow(
	ev *benefitsv1.BenefitEvent,
	sku string,
	customerID string,
	tenantID string,
	environmentID string,
) *domainBenefit.BenefitLedger {
	now := time.Now().UTC()
	row := &domainBenefit.BenefitLedger{
		ID:             types.GenerateUUID(),
		EventID:        ev.GetEventId(),
		SubscriptionID: ev.GetSubscriptionId(),
		CustomerID:     customerID,
		SKU:            sku,
		CycleID:        ev.GetCycleId(),
		Category:       strings.TrimPrefix(ev.GetCategory().String(), categoryEnumPrefix),
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

func (s *benefitConsumptionService) validateEvent(
	ctx context.Context,
	ev *benefitsv1.BenefitEvent,
) (sku string, customerID string, dropReason string, retryErr error) {

	cust, err := s.CustomerRepo.GetByLookupKey(ctx, ev.GetExternalCustomerId())
	if err != nil {
		if ierr.IsNotFound(err) {
			return "", "", "customer not found for external_customer_id", nil
		}
		return "", "", "", ierr.WithError(err).WithHint("customer lookup failed").Mark(ierr.ErrDatabase)
	}

	sub, err := s.SubRepo.Get(ctx, ev.GetSubscriptionId())
	if err != nil {
		if ierr.IsNotFound(err) {
			return "", "", "subscription not found", nil
		}
		return "", "", "", ierr.WithError(err).WithHint("subscription lookup failed").Mark(ierr.ErrDatabase)
	}
	if sub.CustomerID != cust.ID {
		return "", "", "subscription does not belong to customer", nil
	}
	if sub.SubscriptionStatus != types.SubscriptionStatusActive {
		return "", "", "subscription is not active", nil
	}
	if sub.Sku == nil || *sub.Sku == "" {
		return "", "", "subscription has no sku", nil
	}

	if reason, rErr := s.assertPlanGrantsFeature(ctx, sub.PlanID, ev.GetFeatureId()); rErr != nil {
		return "", "", "", rErr
	} else if reason != "" {
		return "", "", reason, nil
	}

	inv, err := s.InvoiceRepo.Get(ctx, ev.GetCycleId())
	if err != nil {
		if ierr.IsNotFound(err) {
			return "", "", "invoice (cycle_id) not found", nil
		}
		return "", "", "", ierr.WithError(err).WithHint("invoice lookup failed").Mark(ierr.ErrDatabase)
	}
	if inv.SubscriptionID == nil || *inv.SubscriptionID != ev.GetSubscriptionId() {
		return "", "", "invoice does not belong to subscription", nil
	}
	if inv.CustomerID != cust.ID {
		return "", "", "invoice does not belong to customer", nil
	}
	if inv.InvoiceStatus != types.InvoiceStatusFinalized {
		return "", "", "invoice is not finalized", nil
	}
	if inv.PaymentStatus != types.PaymentStatusSucceeded {
		return "", "", "invoice payment is not succeeded", nil
	}

	return *sub.Sku, cust.ID, "", nil
}

func (s *benefitConsumptionService) assertPlanGrantsFeature(
	ctx context.Context,
	planID string,
	featureID string,
) (dropReason string, err error) {
	planEnts, err := s.EntitlementRepo.ListByPlanIDs(ctx, []string{planID})
	if err != nil {
		return "", ierr.WithError(err).WithHint("plan entitlement lookup failed").Mark(ierr.ErrDatabase)
	}
	for _, e := range planEnts {
		if e != nil && e.FeatureID == featureID && e.IsEnabled {
			return "", nil
		}
	}
	return "plan does not grant this feature", nil
}

func (s *benefitConsumptionService) shouldRetryError(err error) bool {
	if errors.Is(err, ierr.ErrValidation) ||
		errors.Is(err, ierr.ErrNotFound) ||
		errors.Is(err, ierr.ErrAlreadyExists) {
		return false
	}
	return true
}
