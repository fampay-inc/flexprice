package benefit

import "context"

type Repository interface {
	Create(ctx context.Context, b *BenefitLedger) error
	GetAggregatedBenefitsByCategory(ctx context.Context, customerID, product string) ([]*BenefitAggregate, error)
}
