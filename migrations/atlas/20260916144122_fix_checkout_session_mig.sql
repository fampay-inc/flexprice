-- atlas:txmode none

-- Drop index "idx_checkout_session_expiry" from table: "checkout_sessions"
DROP INDEX CONCURRENTLY "idx_checkout_session_expiry";
-- Drop index "idx_checkout_session_idempotency_key_active" from table: "checkout_sessions"
DROP INDEX CONCURRENTLY "idx_checkout_session_idempotency_key_active";
-- Create index "idx_checkout_session_expiry" to table: "checkout_sessions"
CREATE INDEX CONCURRENTLY "idx_checkout_session_expiry" ON "checkout_sessions" ("expires_at") WHERE ((checkout_status)::text = ANY (ARRAY['initiated'::text, 'pending'::text]));
-- Create index "idx_checkout_session_idempotency_key_active" to table: "checkout_sessions"
CREATE UNIQUE INDEX CONCURRENTLY "idx_checkout_session_idempotency_key_active" ON "checkout_sessions" ("tenant_id", "environment_id", "idempotency_key") WHERE ((idempotency_key IS NOT NULL) AND ((checkout_status)::text = ANY (ARRAY['initiated'::text, 'pending'::text])));
