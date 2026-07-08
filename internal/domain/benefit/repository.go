package benefit

import "context"

type Repository interface {
	Create(ctx context.Context, b *BenefitLedger) error
}
