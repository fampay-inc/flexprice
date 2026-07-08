package dto

type BenefitAggregateResponse struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Total    int64  `json:"total"`
}
