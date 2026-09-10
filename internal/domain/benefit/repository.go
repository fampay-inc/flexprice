package benefit

import "context"

type Repository interface {
	Create(ctx context.Context, b *BenefitLedger) error

	GetGrantByEventID(ctx context.Context, eventID string) (*BenefitLedger, error)

	UpdateReversedValue(ctx context.Context, product, eventID string, reversedValue int) error

	GetBenefitTypeAggregates(ctx context.Context, customerID, product string) ([]*BenefitTypeAggregate, error)
}
