package dto

type BenefitItem struct {
	Type      string `json:"type"`
	Value     int64  `json:"value"`
	Frequency int64  `json:"frequency"`
}

type BenefitAggregateResponse struct {
	Category string        `json:"category,omitempty"`
	Benefits []BenefitItem `json:"benefits,omitempty"`
}
