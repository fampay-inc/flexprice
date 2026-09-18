-- atlas:txmode none

-- Create index "idx_subscription_tenant_env_lookup_key_unique" to table: "subscriptions"
CREATE UNIQUE INDEX CONCURRENTLY "idx_subscription_tenant_env_lookup_key_unique" ON "subscriptions" ("tenant_id", "environment_id", "lookup_key") WHERE ((lookup_key IS NOT NULL) AND ((lookup_key)::text <> ''::text) AND ((status)::text = 'published'::text));
