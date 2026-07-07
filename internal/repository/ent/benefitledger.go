package ent

import (
	"context"
	"errors"

	"github.com/flexprice/flexprice/ent"
	domainBenefit "github.com/flexprice/flexprice/internal/domain/benefit"
	ierr "github.com/flexprice/flexprice/internal/errors"
	"github.com/flexprice/flexprice/internal/logger"
	"github.com/flexprice/flexprice/internal/postgres"
	"github.com/flexprice/flexprice/internal/types"
	"github.com/lib/pq"
)

type benefitLedgerRepository struct {
	client postgres.IClient
	log    *logger.Logger
}


func NewBenefitLedgerRepository(client postgres.IClient, log *logger.Logger) domainBenefit.Repository {
	return &benefitLedgerRepository{
		client: client,
		log:    log,
	}
}

func (r *benefitLedgerRepository) Create(ctx context.Context, b *domainBenefit.BenefitLedger) error {
	client := r.client.Writer(ctx)

	span := StartRepositorySpan(ctx, "benefit_ledger", "create", map[string]interface{}{
		"event_id":        b.EventID,
		"subscription_id": b.SubscriptionID,
		"sku":             b.SKU,
		"category":        b.Category,
	})
	defer FinishSpan(span)

	if b.EnvironmentID == "" {
		b.EnvironmentID = types.GetEnvironmentID(ctx)
	}

	_, err := client.BenefitLedger.Create().
		SetID(b.ID).
		SetTenantID(b.TenantID).
		SetEnvironmentID(b.EnvironmentID).
		SetEventID(b.EventID).
		SetSubscriptionID(b.SubscriptionID).
		SetCustomerID(b.CustomerID).
		SetSku(b.SKU).
		SetCycleID(b.CycleID).
		SetCategory(b.Category).
		SetFeatureID(b.FeatureID).
		SetValue(b.Value).
		SetEventTimestamp(b.EventTimestamp).
		SetStatus(string(types.StatusPublished)).
		SetCreatedAt(b.CreatedAt).
		SetUpdatedAt(b.UpdatedAt).
		SetCreatedBy(b.CreatedBy).
		SetUpdatedBy(b.UpdatedBy).
		Save(ctx)

	if err != nil {
		SetSpanError(span, err)

		if ent.IsConstraintError(err) {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) {
				return ierr.WithError(err).
					WithHint("Benefit event already recorded").
					WithReportableDetails(map[string]any{
						"event_id": b.EventID,
					}).
					Mark(ierr.ErrAlreadyExists)
			}
		}
		return ierr.WithError(err).
			WithHint("Failed to insert benefit ledger row").
			Mark(ierr.ErrDatabase)
	}

	return nil
}
