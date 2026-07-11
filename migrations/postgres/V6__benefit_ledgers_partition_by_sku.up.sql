CREATE TABLE benefit_ledgers (
    id              varchar(50)  NOT NULL,
    tenant_id       varchar(50)  NOT NULL,
    status          varchar(20)  NOT NULL DEFAULT 'granted',
    created_at      timestamptz  NOT NULL,
    updated_at      timestamptz  NOT NULL,
    created_by      varchar,
    updated_by      varchar,
    environment_id  varchar(50)  DEFAULT '',
    event_id        varchar(255) NOT NULL,
    subscription_id uuid         NOT NULL,
    customer_id     uuid         NOT NULL,
    product         varchar(50)  NOT NULL,
    cycle_id        uuid         NOT NULL,
    category        varchar(50),
    feature_id      uuid         NOT NULL,
    value           bigint       NOT NULL,
    event_timestamp timestamptz  NOT NULL,
    PRIMARY KEY (id, product),
    CONSTRAINT uq_benefit_ledger_event_id UNIQUE (product, event_id)
) PARTITION BY LIST (product);

CREATE TABLE benefit_ledgers_limitless PARTITION OF benefit_ledgers FOR VALUES IN ('limitless');
CREATE TABLE benefit_ledgers_default   PARTITION OF benefit_ledgers DEFAULT;

CREATE INDEX idx_benefit_ledger_customer_cycle ON benefit_ledgers (tenant_id, environment_id, customer_id, cycle_id);
