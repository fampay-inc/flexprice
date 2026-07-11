package dto

type BenefitAggregateResponse struct {
	Category  string `json:"category,omitempty"`
	FeatureID string `json:"feature_id,omitempty"`
	Total     int64  `json:"total"`
}
