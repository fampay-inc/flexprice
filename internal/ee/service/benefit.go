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

	aggregates, err := s.BenefitLedgerRepo.GetBenefitTypeAggregates(ctx, cust.ID, product)
	if err != nil {
		return nil, err
	}

	index := make(map[string]int)
	response := make([]*dto.BenefitAggregateResponse, 0)
	for _, agg := range aggregates {
		if agg.Net <= 0 {
			continue
		}
		benefitType := agg.BenefitType
		i, seen := index[agg.Category]
		if !seen {
			response = append(response, &dto.BenefitAggregateResponse{Category: agg.Category})
			i = len(response) - 1
			index[agg.Category] = i
		}
		response[i].Benefits = append(response[i].Benefits, dto.BenefitItem{
			Type:  benefitType,
			Value: agg.Net,
		})
		response[i].Total += agg.Net
	}
	return response, nil
}
