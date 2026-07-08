package benefit

import "context"

type Repository interface {
	Create(ctx context.Context, b *BenefitLedger) error
	GetAggregatedBenefits(ctx context.Context, customerID, sku string) ([]*BenefitAggregate, error)
}
