-- atlas:txmode none

-- Create index "idx_subscriptions_unique_product_live" to table: "subscriptions"
CREATE UNIQUE INDEX CONCURRENTLY "idx_subscriptions_unique_product_live" ON "subscriptions" ("tenant_id", "environment_id", "customer_id", "product") WHERE (((subscription_status)::text = ANY (ARRAY[('active'::character varying)::text, ('trialing'::character varying)::text, ('paused'::character varying)::text])) AND ((status)::text = 'published'::text) AND (product IS NOT NULL));
