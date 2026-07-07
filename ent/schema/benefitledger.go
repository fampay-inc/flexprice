package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/flexprice/flexprice/ent/schema/mixin"
)

type BenefitLedger struct {
	ent.Schema
}

func (BenefitLedger) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.BaseMixin{},
		mixin.EnvironmentMixin{},
	}
}

func (BenefitLedger) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			SchemaType(map[string]string{
				"postgres": "varchar(50)",
			}).
			Unique().
			Immutable(),

		field.String("event_id").
			SchemaType(map[string]string{
				"postgres": "varchar(255)",
			}).
			NotEmpty().
			Immutable(),

		field.String("subscription_id").
			SchemaType(map[string]string{
				"postgres": "uuid",
			}).
			NotEmpty().
			Immutable(),

		field.String("customer_id").
			SchemaType(map[string]string{
				"postgres": "uuid",
			}).
			NotEmpty().
			Immutable(),

		field.String("sku").
			SchemaType(map[string]string{
				"postgres": "varchar(50)",
			}).
			NotEmpty().
			Immutable(),

		field.String("cycle_id").
			SchemaType(map[string]string{
				"postgres": "uuid",
			}).
			NotEmpty().
			Immutable(),

		field.String("category").
			SchemaType(map[string]string{
				"postgres": "varchar(50)",
			}).
			NotEmpty().
			Immutable(),

		field.String("feature_id").
			SchemaType(map[string]string{
				"postgres": "uuid",
			}).
			Optional().
			Immutable(),

		field.Int("value").
			Immutable(),

		field.Time("event_timestamp").
			Immutable(),
	}
}

func (BenefitLedger) Edges() []ent.Edge {
	return nil
}

func (BenefitLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sku", "event_id").
			Unique().
			StorageKey("uq_benefit_ledger_event_id"),
		index.Fields("tenant_id", "environment_id", "customer_id", "cycle_id").
			StorageKey("idx_benefit_ledger_customer_cycle"),
	}
}
