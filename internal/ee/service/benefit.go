package service

import (
	"context"

	"github.com/flexprice/flexprice/internal/api/dto"
)

type BenefitService interface {
	GetBenefits(ctx context.Context, externalCustomerID, product string) ([]*dto.BenefitAggregateResponse, error)
}

type benefitService struct {
	ServiceParams
}

func NewBenefitService(params ServiceParams) BenefitService {
	return &benefitService{ServiceParams: params}
}

func (s *benefitService) GetBenefits(ctx context.Context, externalCustomerID, product string) ([]*dto.BenefitAggregateResponse, error) {
	cust, err := s.CustomerRepo.GetByLookupKey(ctx, externalCustomerID)
	if err != nil {
		return nil, err
	}

	aggregates, err := s.BenefitLedgerRepo.GetAggregatedBenefitsByCategory(ctx, cust.ID, product)
	if err != nil {
		return nil, err
	}

	response := make([]*dto.BenefitAggregateResponse, 0, len(aggregates))
	for _, agg := range aggregates {
		response = append(response, &dto.BenefitAggregateResponse{
			Category: agg.Category,
			Total:    agg.Total,
		})
	}

	return response, nil
}
