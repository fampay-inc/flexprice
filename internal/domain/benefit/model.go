package benefit

import (
	"time"
	"github.com/flexprice/flexprice/internal/types"
)

type BenefitLedger struct {
	ID             string    `db:"id" json:"id"`
	EventID        string    `db:"event_id" json:"event_id"`
	SubscriptionID string    `db:"subscription_id" json:"subscription_id"`
	CustomerID     string    `db:"customer_id" json:"customer_id"`
	Product        string    `db:"product" json:"product"`
	CycleID        string    `db:"cycle_id" json:"cycle_id"`
	Category       string    `db:"category" json:"category"`
	FeatureID      string    `db:"feature_id" json:"feature_id"`
	Value          int       `db:"value" json:"value"`
	EventTimestamp time.Time `db:"event_timestamp" json:"event_timestamp"`
	EnvironmentID  string    `db:"environment_id" json:"environment_id"`
	types.BaseModel
}

type BenefitAggregate struct {
	Category  string `db:"category" json:"category"`
	FeatureID string `db:"feature_id" json:"feature_id"`
	Total     int64  `db:"total" json:"total"`
}
