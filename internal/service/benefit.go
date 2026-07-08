package service

import (
	"context"

	"github.com/flexprice/flexprice/internal/api/dto"
	featureDomain "github.com/flexprice/flexprice/internal/domain/feature"
	ierr "github.com/flexprice/flexprice/internal/errors"
)

type BenefitService interface {
	GetBenefitsBySKU(ctx context.Context, externalCustomerID, sku string) ([]*dto.BenefitAggregateResponse, error)
}

type benefitService struct {
	ServiceParams
}

func NewBenefitService(params ServiceParams) BenefitService {
	return &benefitService{ServiceParams: params}
}

func (s *benefitService) GetBenefitsBySKU(ctx context.Context, externalCustomerID, sku string) ([]*dto.BenefitAggregateResponse, error) {
	if externalCustomerID == "" {
		return nil, ierr.NewError("external_customer_id is required").
			WithHint("external_customer_id is required").
			Mark(ierr.ErrValidation)
	}
	if sku == "" {
		return nil, ierr.NewError("sku is required").
			WithHint("sku is required").
			Mark(ierr.ErrValidation)
	}

	cust, err := s.CustomerRepo.GetByLookupKey(ctx, externalCustomerID)
	if err != nil {
		return nil, err
	}

	aggregates, err := s.BenefitLedgerRepo.GetAggregatedBenefits(ctx, cust.ID, sku)
	if err != nil {
		return nil, err
	}

	featureIDs := make([]string, 0, len(aggregates))
	for _, agg := range aggregates {
		featureIDs = append(featureIDs, agg.FeatureID)
	}

	featureMap := make(map[string]*featureDomain.Feature, len(featureIDs))
	if len(featureIDs) > 0 {
		features, err := s.FeatureRepo.ListByIDs(ctx, featureIDs)
		if err != nil {
			return nil, err
		}
		for _, f := range features {
			featureMap[f.ID] = f
		}
	}

	response := make([]*dto.BenefitAggregateResponse, 0, len(aggregates))
	for _, agg := range aggregates {
		f, ok := featureMap[agg.FeatureID]
		if !ok {
			s.Logger.Warnw("feature not found for benefit aggregate",
				"feature_id", agg.FeatureID,
				"sku", sku,
			)
			continue
		}
		response = append(response, &dto.BenefitAggregateResponse{
			Name:     f.Name,
			Slug:     f.LookupKey,
			Metadata: f.Metadata,
			Total:    agg.Total,
		})
	}

	return response, nil
}
