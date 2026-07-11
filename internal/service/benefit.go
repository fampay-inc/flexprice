package service

import (
	"context"

	"github.com/flexprice/flexprice/internal/api/dto"
	ierr "github.com/flexprice/flexprice/internal/errors"
)

const benefitGroupByCategory = "category"

type BenefitService interface {
	GetBenefits(ctx context.Context, externalCustomerID, product, groupBy string) ([]*dto.BenefitAggregateResponse, error)
}

type benefitService struct {
	ServiceParams
}

func NewBenefitService(params ServiceParams) BenefitService {
	return &benefitService{ServiceParams: params}
}

func (s *benefitService) GetBenefits(ctx context.Context, externalCustomerID, product, groupBy string) ([]*dto.BenefitAggregateResponse, error) {
	if groupBy != "" && groupBy != benefitGroupByCategory {
		return nil, ierr.NewError("unsupported group_by value").
			WithHint("group_by must be 'category'").
			Mark(ierr.ErrValidation)
	}

	cust, err := s.CustomerRepo.GetByLookupKey(ctx, externalCustomerID)
	if err != nil {
		return nil, err
	}

	aggregates, err := s.BenefitLedgerRepo.GetAggregatedBenefits(ctx, cust.ID, product, groupBy)
	if err != nil {
		return nil, err
	}

	byCategory := groupBy == benefitGroupByCategory

	response := make([]*dto.BenefitAggregateResponse, 0, len(aggregates))
	for _, agg := range aggregates {
		if byCategory {
			response = append(response, &dto.BenefitAggregateResponse{
				Category: agg.Category,
				Total:    agg.Total,
			})
		} else {
			response = append(response, &dto.BenefitAggregateResponse{
				FeatureID: agg.FeatureID,
				Total:     agg.Total,
			})
		}
	}

	return response, nil
}
