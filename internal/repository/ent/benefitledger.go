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
		"product":         b.Product,
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
		SetProduct(b.Product).
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

func (r *benefitLedgerRepository) GetAggregatedBenefits(ctx context.Context, customerID, sku string) ([]*domainBenefit.BenefitAggregate, error) {
	tenantID := types.GetTenantID(ctx)
	environmentID := types.GetEnvironmentID(ctx)

	span := StartRepositorySpan(ctx, "benefit_ledger", "get_aggregated_benefits", map[string]interface{}{
		"customer_id": customerID,
		"sku":         sku,
	})
	defer FinishSpan(span)

	query := `
		SELECT feature_id, COALESCE(SUM(value), 0)::bigint AS total
		FROM benefit_ledgers
		WHERE product = $1
			AND tenant_id = $2
			AND environment_id = $3
			AND customer_id = $4
		GROUP BY feature_id`

	rows, err := r.client.Reader(ctx).QueryContext(ctx, query, sku, tenantID, environmentID, customerID)
	if err != nil {
		SetSpanError(span, err)
		return nil, ierr.WithError(err).
			WithHint("Failed to aggregate benefit ledger rows").
			Mark(ierr.ErrDatabase)
	}
	defer rows.Close()

	results := make([]*domainBenefit.BenefitAggregate, 0)
	for rows.Next() {
		agg := &domainBenefit.BenefitAggregate{}
		if err := rows.Scan(&agg.FeatureID, &agg.Total); err != nil {
			SetSpanError(span, err)
			return nil, ierr.WithError(err).
				WithHint("Failed to scan benefit aggregate row").
				Mark(ierr.ErrDatabase)
		}
		results = append(results, agg)
	}

	if err := rows.Err(); err != nil {
		SetSpanError(span, err)
		return nil, ierr.WithError(err).
			WithHint("Failed to iterate benefit aggregate rows").
			Mark(ierr.ErrDatabase)
	}

	SetSpanSuccess(span)
	return results, nil
}
