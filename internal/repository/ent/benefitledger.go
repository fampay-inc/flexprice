package ent

import (
	"context"
	"errors"

	"github.com/flexprice/flexprice/ent"
	"github.com/flexprice/flexprice/ent/benefitledger"
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
	span := StartRepositorySpan(ctx, "benefit_ledger", "create", map[string]interface{}{
		"event_id":        b.EventID,
		"subscription_id": b.SubscriptionID,
		"product":         b.Product,
		"category":        b.Category,
		"entry_type":      b.EntryType,
	})
	defer FinishSpan(span)

	if b.EnvironmentID == "" {
		b.EnvironmentID = types.GetEnvironmentID(ctx)
	}

	entryType := string(b.EntryType)
	q := r.client.Writer(ctx).BenefitLedger.Create().
		SetID(b.ID).
		SetTenantID(b.TenantID).
		SetEnvironmentID(b.EnvironmentID).
		SetEventID(b.EventID).
		SetSubscriptionID(b.SubscriptionID).
		SetCustomerID(b.CustomerID).
		SetProduct(b.Product).
		SetCategory(b.Category).
		SetFeatureID(b.FeatureID).
		SetValue(b.Value).
		SetEventTimestamp(b.EventTimestamp).
		SetStatus(string(types.StatusPublished)).
		SetCreatedAt(b.CreatedAt).
		SetUpdatedAt(b.UpdatedAt).
		SetCreatedBy(b.CreatedBy).
		SetUpdatedBy(b.UpdatedBy).
		SetEntryType(entryType).
		SetBenefitType(b.BenefitType).
		SetReversedValue(b.ReversedValue).
		SetOriginalEventID(b.OriginalEventID)

	_, err := q.Save(ctx)
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

func (r *benefitLedgerRepository) GetGrantByEventID(ctx context.Context, eventID string) (*domainBenefit.BenefitLedger, error) {
	tenantID := types.GetTenantID(ctx)
	environmentID := types.GetEnvironmentID(ctx)

	span := StartRepositorySpan(ctx, "benefit_ledger", "get_grant_by_event_id", map[string]interface{}{
		"event_id": eventID,
	})
	defer FinishSpan(span)

	row, err := r.client.Reader(ctx).BenefitLedger.Query().
		Where(
			benefitledger.EventID(eventID),
			benefitledger.TenantID(tenantID),
			benefitledger.EnvironmentID(environmentID),
			benefitledger.Or(
				benefitledger.EntryTypeIsNil(),
				benefitledger.EntryTypeEQ(""),
				benefitledger.EntryTypeEQ(string(domainBenefit.EntryTypeGrant)),
			),
		).
		First(ctx)
	if err != nil {
		SetSpanError(span, err)
		if ent.IsNotFound(err) {
			return nil, ierr.NewError("original grant not found").
				WithHint("No grant row with entry_type='grant' found for event_id").
				Mark(ierr.ErrNotFound)
		}
		return nil, ierr.WithError(err).
			WithHint("Failed to fetch grant row").
			Mark(ierr.ErrDatabase)
	}

	b := &domainBenefit.BenefitLedger{
		ID:             row.ID,
		Product:        row.Product,
		SubscriptionID: row.SubscriptionID,
		CustomerID:     row.CustomerID,
		FeatureID:      row.FeatureID,
		BenefitType:    row.BenefitType,
		Value:          row.Value,
		ReversedValue:  row.ReversedValue,
	}
	if row.EntryType != "" {
		b.EntryType = domainBenefit.EntryType(row.EntryType)
	} else {
		b.EntryType = domainBenefit.EntryTypeGrant
	}

	SetSpanSuccess(span)
	return b, nil
}

func (r *benefitLedgerRepository) UpdateReversedValue(ctx context.Context, product, eventID string, reversedValue int) error {
	tenantID := types.GetTenantID(ctx)
	environmentID := types.GetEnvironmentID(ctx)

	span := StartRepositorySpan(ctx, "benefit_ledger", "update_reversed_value", map[string]interface{}{
		"event_id":      eventID,
		"product":       product,
		"reversedValue": reversedValue,
	})
	defer FinishSpan(span)

	n, err := r.client.Writer(ctx).BenefitLedger.Update().
		Where(
			benefitledger.Product(product),
			benefitledger.EventID(eventID),
			benefitledger.TenantID(tenantID),
			benefitledger.EnvironmentID(environmentID),
			benefitledger.Or(
				benefitledger.EntryTypeIsNil(),
				benefitledger.EntryTypeEQ(""),
				benefitledger.EntryTypeEQ(string(domainBenefit.EntryTypeGrant)),
			),
		).
		AddReversedValue(reversedValue).
		Save(ctx)
	if err != nil {
		SetSpanError(span, err)
		return ierr.WithError(err).
			WithHint("Failed to update reversed_value on grant row").
			Mark(ierr.ErrDatabase)
	}
	if n == 0 {
		SetSpanError(span, ierr.ErrNotFound)
		return ierr.NewError("grant row not found for reversed_value update").
			WithHint("No matching grant row for product + event_id").
			Mark(ierr.ErrNotFound)
	}

	SetSpanSuccess(span)
	return nil
}

func (r *benefitLedgerRepository) GetBenefitTypeAggregates(ctx context.Context, customerID, product string) ([]*domainBenefit.BenefitTypeAggregate, error) {
	tenantID := types.GetTenantID(ctx)
	environmentID := types.GetEnvironmentID(ctx)

	span := StartRepositorySpan(ctx, "benefit_ledger", "get_benefit_type_aggregates", map[string]interface{}{
		"customer_id": customerID,
		"product":     product,
	})
	defer FinishSpan(span)

	query := `
		SELECT
			COALESCE(category, '') AS category,
			COALESCE(benefit_type, '') AS benefit_type,
			SUM(value - reversed_value) AS net,
			COUNT(CASE WHEN (value - reversed_value) > 0 THEN 1 END) AS frequency
		FROM benefit_ledgers
		WHERE
			tenant_id = $1
			AND environment_id = $2
			AND customer_id = $3
			AND product = $4
			AND (entry_type IS NULL OR entry_type = '' OR entry_type = 'grant')
		GROUP BY category, benefit_type
	`

	dbRows, err := r.client.Reader(ctx).QueryContext(ctx, query, tenantID, environmentID, customerID, product)
	if err != nil {
		SetSpanError(span, err)
		return nil, ierr.WithError(err).
			WithHint("Failed to query benefit type aggregates").
			Mark(ierr.ErrDatabase)
	}
	defer dbRows.Close()

	results := make([]*domainBenefit.BenefitTypeAggregate, 0)
	for dbRows.Next() {
		agg := &domainBenefit.BenefitTypeAggregate{}
		if err := dbRows.Scan(&agg.Category, &agg.BenefitType, &agg.Net, &agg.Frequency); err != nil {
			SetSpanError(span, err)
			return nil, ierr.WithError(err).
				WithHint("Failed to scan benefit type aggregate row").
				Mark(ierr.ErrDatabase)
		}
		results = append(results, agg)
	}
	if err := dbRows.Err(); err != nil {
		SetSpanError(span, err)
		return nil, ierr.WithError(err).
			WithHint("Error iterating benefit type aggregate rows").
			Mark(ierr.ErrDatabase)
	}

	SetSpanSuccess(span)
	return results, nil
}
