ALTER TABLE benefit_ledgers
  ADD COLUMN IF NOT EXISTS benefit_type      varchar(50),
  ADD COLUMN IF NOT EXISTS entry_type        varchar(20),
  ADD COLUMN IF NOT EXISTS original_event_id varchar(255),
  ADD COLUMN IF NOT EXISTS reversed_value    bigint NOT NULL DEFAULT 0;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_benefit_ledger_original_event
  ON benefit_ledgers (product, original_event_id)
  WHERE original_event_id IS NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_class c
    JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relname = 'benefit_ledgers_flex'
      AND n.nspname = 'public'
  ) THEN
    CREATE TABLE benefit_ledgers_flex PARTITION OF benefit_ledgers FOR VALUES IN ('flex');
  END IF;
END $$;
