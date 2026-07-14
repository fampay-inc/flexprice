ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS product varchar(255) NOT NULL;

CREATE INDEX IF NOT EXISTS subscriptions_product_idx ON subscriptions (product);

CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_tenant_env_customer_product_active_idx
    ON subscriptions (tenant_id, environment_id, customer_id, product)
    WHERE subscription_status = 'active' AND status = 'published';
