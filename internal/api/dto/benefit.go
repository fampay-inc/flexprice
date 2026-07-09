package dto

import "github.com/flexprice/flexprice/internal/types"

type BenefitAggregateResponse struct {
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	Metadata types.Metadata `json:"metadata"`
	Total    int64          `json:"total"`
}
