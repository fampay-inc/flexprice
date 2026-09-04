package dto

type BenefitAggregateResponse struct {
	Category string `json:"category,omitempty"`
	Total    int64  `json:"total"`
}
