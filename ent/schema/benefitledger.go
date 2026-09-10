package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
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

		field.String("product").
			SchemaType(map[string]string{
				"postgres": "varchar(50)",
			}).
			NotEmpty().
			Immutable(),

		field.String("category").
			SchemaType(map[string]string{
				"postgres": "varchar(50)",
			}).
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

		field.String("benefit_type").
			SchemaType(map[string]string{
				"postgres": "varchar(50)",
			}).
			Optional().
			Immutable(),

		field.String("entry_type").
			SchemaType(map[string]string{
				"postgres": "varchar(20)",
			}).
			Optional().
			Immutable(),

		field.String("original_event_id").
			SchemaType(map[string]string{
				"postgres": "varchar(255)",
			}).
			Optional().
			Immutable(),

		field.Int("reversed_value").
			Default(0),
	}
}

func (BenefitLedger) Edges() []ent.Edge {
	return nil
}

func (BenefitLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product", "event_id").
			Unique().
			StorageKey("uq_benefit_ledger_event_id"),

		index.Fields("tenant_id", "environment_id", "customer_id").
			StorageKey("idx_benefit_ledger_customer"),

		index.Fields("product", "original_event_id").
			StorageKey("idx_benefit_ledger_original_event").
			Annotations(entsql.IndexWhere("original_event_id IS NOT NULL")),
	}
}
