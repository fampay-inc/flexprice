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
	"github.com/flexprice/flexprice/internal/metrics"
	"github.com/flexprice/flexprice/internal/pubsub"
	"github.com/flexprice/flexprice/internal/pubsub/kafka"
	pubsubRouter "github.com/flexprice/flexprice/internal/pubsub/router"
	"github.com/flexprice/flexprice/internal/tracing"
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
	pubSub              pubsub.PubSub
	sentryService       *tracing.Service
	subscriptionService SubscriptionService
}

func NewBenefitConsumptionService(
	params ServiceParams,
	sentryService *tracing.Service,
) BenefitConsumptionService {
	s := &benefitConsumptionService{
		ServiceParams:       params,
		sentryService:       sentryService,
		subscriptionService: NewSubscriptionService(params),
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
		"",
		s.pubSub,
		s.processMessage,
		throttle.Middleware,
	)

	s.Logger.Infow("registered benefit consumption handler",
		"topic", cfg.BenefitEvents.Topic,
		"rate_limit", cfg.BenefitEvents.RateLimit,
	)
}

func (s *benefitConsumptionService) processMessage(ctx context.Context, msg *message.Message) error {
	var ev benefitsv1.BenefitEvent
	if err := proto.Unmarshal(msg.Payload, &ev); err != nil {
		s.Logger.Errorw("failed to unmarshal benefit event proto",
			"error", err,
			"payload_len", len(msg.Payload),
		)
		s.sentryService.CaptureException(ctx, err)
		return nil
	}

	entryType := ev.GetEntryType()
	if entryType == benefitsv1.EntryType_ENTRY_TYPE_UNSPECIFIED {
		entryType = benefitsv1.EntryType_GRANT
	}

	tenantID := s.Config.Billing.TenantID
	if tenantID == "" {
		s.Logger.Errorw("billing.tenant_id is not configured; cannot process benefit events",
			"event_id", ev.GetEventId(),
		)
		return nil
	}

	ctx = context.WithValue(ctx, types.CtxTenantID, tenantID)
	if environmentID := s.Config.Billing.EnvironmentID; environmentID != "" {
		ctx = context.WithValue(ctx, types.CtxEnvironmentID, environmentID)
	}

	switch entryType {
	case benefitsv1.EntryType_GRANT:
		return s.processGrantEvent(ctx, &ev, tenantID)
	case benefitsv1.EntryType_REVERSAL:
		return s.processReversalEvent(ctx, &ev, tenantID)
	default:
		s.Logger.Warnw("unknown entry_type, dropping",
			"entry_type", entryType,
			"event_id", ev.GetEventId(),
		)
		return nil
	}
}

func (s *benefitConsumptionService) processGrantEvent(ctx context.Context, event *benefitsv1.BenefitEvent, tenantID string) error {
	if err := validateGrantFields(event); err != nil {
		s.Logger.Warnw("dropping invalid benefit grant event",
			"reason", err.Error(),
			"event_id", event.GetEventId(),
			"username", event.GetUsername(),
		)
		return nil
	}

	s.Logger.Infow("processing benefit grant event",
		"event_id", event.GetEventId(),
		"username", event.GetUsername(),
	)

	validated, err := s.validateEvent(ctx, event)
	if err != nil {
		if isDropEvent(err) {
			s.Logger.Warnw("dropping invalid benefit grant event",
				"reason", err.Error(),
				"event_id", event.GetEventId(),
				"username", event.GetUsername(),
				"subscription_id", event.GetSubscriptionId(),
			)
			return nil
		}
		s.Logger.Errorw("benefit grant event validation errored, will retry",
			"error", err,
			"event_id", event.GetEventId(),
			"username", event.GetUsername(),
		)
		return ierr.WithError(err).
			WithHint("Failed to validate benefit grant event").
			Mark(ierr.ErrSystem)
	}

	row := toLedgerRow(event, validated, event.GetBenefitType(), tenantID, s.Config.Billing.EnvironmentID)

	if err := s.BenefitLedgerRepo.Create(ctx, row); err != nil {
		if ierr.IsAlreadyExists(err) {
			s.Logger.Info(ctx, "duplicate benefit grant event ignored",
				"event_id", event.GetEventId(),
				"username", event.GetUsername(),
			)
			metrics.BenefitLedgerDuplicatesTotal.Inc()
			return nil
		}

		s.Logger.Errorw("failed to store benefit grant event",
			"error", err,
			"event_id", event.GetEventId(),
			"username", event.GetUsername(),
		)

		if !s.shouldRetryError(err) {
			return nil
		}
		return ierr.WithError(err).
			WithHint("Failed to store benefit grant event").
			Mark(ierr.ErrSystem)
	}

	s.Logger.Debugw("stored benefit grant event",
		"event_id", row.EventID,
		"username", event.GetUsername(),
	)

	return nil
}

func (s *benefitConsumptionService) processReversalEvent(ctx context.Context, event *benefitsv1.BenefitEvent, tenantID string) error {
	if err := validateReversalFields(event); err != nil {
		s.Logger.Warnw("dropping invalid benefit reversal event",
			"reason", err.Error(),
			"event_id", event.GetEventId(),
		)
		return nil
	}

	originalEventID := event.GetOriginalEventId()

	s.Logger.Infow("processing benefit reversal event",
		"event_id", event.GetEventId(),
		"original_event_id", originalEventID,
	)

	grant, err := s.BenefitLedgerRepo.GetGrantByEventID(ctx, originalEventID)
	if err != nil {
		if ierr.IsNotFound(err) {
			s.Logger.Warnw("dropping reversal: original grant not found",
				"event_id", event.GetEventId(),
				"original_event_id", originalEventID,
			)
			return nil
		}
		s.Logger.Errorw("failed to look up original grant for reversal, will retry",
			"error", err,
			"event_id", event.GetEventId(),
			"original_event_id", originalEventID,
		)
		return ierr.WithError(err).
			WithHint("Failed to fetch original grant for reversal").
			Mark(ierr.ErrSystem)
	}

	incomingValue := int(event.GetValue())
	if incomingValue != grant.Value {
		s.Logger.Errorw("dropping reversal: value mismatch, only full reversal supported",
			"event_id", event.GetEventId(),
			"original_event_id", originalEventID,
			"incoming_value", incomingValue,
			"grant_value", grant.Value,
		)
		return nil
	}

	reversalRow := toReversalLedgerRow(event, grant, tenantID, s.Config.Billing.EnvironmentID)

	if err := s.DB.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.BenefitLedgerRepo.Create(txCtx, reversalRow); err != nil {
			return err
		}
		return s.BenefitLedgerRepo.UpdateReversedValue(txCtx, grant.Product, originalEventID, incomingValue)
	}); err != nil {
		if ierr.IsAlreadyExists(err) {
			s.Logger.Info(ctx, "duplicate benefit reversal event ignored",
				"event_id", event.GetEventId(),
				"original_event_id", originalEventID,
			)
			metrics.BenefitLedgerDuplicatesTotal.Inc()
			return nil
		}

		s.Logger.Errorw("failed to store benefit reversal event",
			"error", err,
			"event_id", event.GetEventId(),
			"original_event_id", originalEventID,
		)

		if !s.shouldRetryError(err) {
			return nil
		}
		return ierr.WithError(err).
			WithHint("Failed to store benefit reversal event").
			Mark(ierr.ErrSystem)
	}

	s.Logger.Debugw("stored benefit reversal event",
		"event_id", reversalRow.EventID,
		"original_event_id", originalEventID,
	)

	return nil
}

func validateGrantFields(ev *benefitsv1.BenefitEvent) error {
	if ev.GetEventId() == "" || ev.GetSubscriptionId() == "" || ev.GetUsername() == "" || ev.GetFeatureId() == "" {
		return drop("missing required fields or fields empty")
	}

	if ev.GetValue() <= 0 {
		return drop("value must be greater than 0")
	}

	for name, val := range map[string]string{
		"subscription_id": ev.GetSubscriptionId(),
		"feature_id":      ev.GetFeatureId(),
	} {
		if _, err := uuid.Parse(val); err != nil {
			return drop(name + " is not a valid UUID")
		}
	}
	return nil
}

func validateReversalFields(ev *benefitsv1.BenefitEvent) error {
	if ev.GetEventId() == "" || ev.GetOriginalEventId() == "" {
		return drop("reversal missing event_id or original_event_id")
	}
	if ev.GetValue() <= 0 {
		return drop("reversal value must be greater than 0")
	}
	return nil
}

func (s *benefitConsumptionService) validateEvent(ctx context.Context, ev *benefitsv1.BenefitEvent) (*eventValidation, error) {
	cust, err := s.CustomerRepo.GetByLookupKey(ctx, ev.GetUsername())
	if err != nil {
		if ierr.IsNotFound(err) {
			return nil, drop("customer not found for username")
		}
		return nil, ierr.WithError(err).WithHint("customer lookup failed").Mark(ierr.ErrDatabase)
	}

	sub, err := s.SubRepo.Get(ctx, ev.GetSubscriptionId())
	if err != nil {
		if ierr.IsNotFound(err) {
			return nil, drop("subscription not found")
		}
		return nil, ierr.WithError(err).WithHint("subscription lookup failed").Mark(ierr.ErrDatabase)
	}

	if sub.CustomerID != cust.ID {
		return nil, drop("subscription does not belong to customer")
	}

	if err := s.validateFeatureEntitlement(ctx, sub.ID, ev.GetFeatureId()); err != nil {
		return nil, err
	}

	return &eventValidation{Product: sub.Product, CustomerID: sub.CustomerID}, nil
}

func (s *benefitConsumptionService) validateFeatureEntitlement(ctx context.Context, subscriptionID, featureID string) error {
	ents, err := s.subscriptionService.GetSubscriptionEntitlements(ctx, subscriptionID)
	if err != nil {
		if ierr.IsNotFound(err) {
			return drop("subscription not found for entitlement lookup")
		}
		return ierr.WithError(err).WithHint("subscription entitlement lookup failed").Mark(ierr.ErrDatabase)
	}
	for _, e := range ents {
		if e != nil && e.FeatureID == featureID && e.IsEnabled {
			return nil
		}
	}
	return drop("subscription does not grant this feature")
}

func toLedgerRow(
	ev *benefitsv1.BenefitEvent,
	v *eventValidation,
	benefitType string,
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
		Category:       strings.ToLower(ev.GetCategory()),
		FeatureID:      ev.GetFeatureId(),
		Value:          int(ev.GetValue()),
		EventTimestamp: time.Unix(ev.GetTimestamp(), 0).UTC(),
		EnvironmentID:  environmentID,
		BenefitType:    benefitType,
		EntryType:      domainBenefit.EntryTypeGrant,
	}
	row.TenantID = tenantID
	row.Status = types.StatusPublished
	row.CreatedAt = now
	row.UpdatedAt = now
	return row
}

func toReversalLedgerRow(
	ev *benefitsv1.BenefitEvent,
	grant *domainBenefit.BenefitLedger,
	tenantID string,
	environmentID string,
) *domainBenefit.BenefitLedger {
	now := time.Now().UTC()
	row := &domainBenefit.BenefitLedger{
		ID:              types.GenerateUUID(),
		EventID:         ev.GetEventId(),
		OriginalEventID: grant.EventID,
		SubscriptionID:  grant.SubscriptionID,
		CustomerID:      grant.CustomerID,
		Product:         grant.Product,
		Category:        grant.Category,
		FeatureID:       grant.FeatureID,
		BenefitType:     grant.BenefitType,
		Value:           grant.Value,
		EventTimestamp:  time.Unix(ev.GetTimestamp(), 0).UTC(),
		EnvironmentID:   environmentID,
		EntryType:       domainBenefit.EntryTypeReversal,
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
