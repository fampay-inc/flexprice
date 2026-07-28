-- Drop index "idx_checkout_session_expiry" from table: "checkout_sessions"
DROP INDEX "idx_checkout_session_expiry";
-- Drop index "idx_checkout_session_idempotency_key_active" from table: "checkout_sessions"
DROP INDEX "idx_checkout_session_idempotency_key_active";
-- Create index "idx_checkout_session_expiry" to table: "checkout_sessions"
CREATE INDEX "idx_checkout_session_expiry" ON "checkout_sessions" ("expires_at") WHERE ((checkout_status)::text = ANY ((ARRAY['initiated'::character varying, 'pending'::character varying])::text[]));
-- Create index "idx_checkout_session_idempotency_key_active" to table: "checkout_sessions"
CREATE UNIQUE INDEX "idx_checkout_session_idempotency_key_active" ON "checkout_sessions" ("tenant_id", "environment_id", "idempotency_key") WHERE ((idempotency_key IS NOT NULL) AND ((checkout_status)::text = ANY ((ARRAY['initiated'::character varying, 'pending'::character varying])::text[])));
-- Modify "plans" table
ALTER TABLE "plans" ADD COLUMN "product" character varying(255) NOT NULL;
-- Modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "product" character varying(255) NOT NULL;
-- Create index "subscription_tenant_id_environment_id_customer_id_product" to table: "subscriptions"
CREATE UNIQUE INDEX "subscription_tenant_id_environment_id_customer_id_product" ON "subscriptions" ("tenant_id", "environment_id", "customer_id", "product") WHERE (((subscription_status)::text = 'active'::text) AND ((status)::text = 'published'::text) AND (product IS NOT NULL));
-- Modify "usage_records" table
ALTER TABLE "usage_records" DROP COLUMN "currency";
-- Modify "workflow_executions" table
ALTER TABLE "workflow_executions" ALTER COLUMN "id" TYPE character varying(50);
