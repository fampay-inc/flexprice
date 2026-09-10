package dto

type BenefitItem struct {
	Type  string `json:"type"`
	Value int64  `json:"value"`
}

type BenefitAggregateResponse struct {
	Category string        `json:"category,omitempty"`
	Benefits []BenefitItem `json:"benefits,omitempty"`
	Total    int64         `json:"total,omitempty"`
}
