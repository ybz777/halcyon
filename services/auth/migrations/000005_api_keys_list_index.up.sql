-- Supports cursor pagination: ORDER BY created_at DESC, id DESC scoped by service account.
CREATE INDEX IF NOT EXISTS idx_api_keys_service_account_created_id
    ON api_keys (service_account_id, created_at DESC, id DESC);
