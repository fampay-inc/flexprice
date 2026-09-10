package benefit

import (
	"time"

	"github.com/flexprice/flexprice/internal/types"
)

type EntryType string

const (
	EntryTypeGrant    EntryType = "grant"
	EntryTypeReversal EntryType = "reversal"
)

type BenefitLedger struct {
	ID              string    `db:"id"                json:"id"`
	EventID         string    `db:"event_id"          json:"event_id"`
	SubscriptionID  string    `db:"subscription_id"   json:"subscription_id"`
	CustomerID      string    `db:"customer_id"       json:"customer_id"`
	Product         string      `db:"product"           json:"product"`
	Category        string      `db:"category"          json:"category"`
	FeatureID       string      `db:"feature_id"        json:"feature_id"`
	Value           int         `db:"value"             json:"value"`
	EventTimestamp  time.Time   `db:"event_timestamp"   json:"event_timestamp"`
	EnvironmentID   string      `db:"environment_id"    json:"environment_id"`
	BenefitType     string `db:"benefit_type"      json:"benefit_type"`
	EntryType       EntryType `db:"entry_type"        json:"entry_type"`
	OriginalEventID string    `db:"original_event_id" json:"original_event_id,omitempty"`
	ReversedValue   int       `db:"reversed_value"    json:"reversed_value"`
	types.BaseModel
}

type BenefitTypeAggregate struct {
	Category    string      `db:"category"     json:"category"`
	BenefitType string `db:"benefit_type" json:"benefit_type"`
	Net         int64       `db:"net"          json:"net"`
}
