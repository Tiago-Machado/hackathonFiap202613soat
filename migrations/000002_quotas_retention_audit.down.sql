-- 000002_quotas_retention_audit.down.sql
DROP TABLE IF EXISTS audit_events;

DROP INDEX IF EXISTS idx_videos_expiry;
ALTER TABLE videos
  DROP COLUMN IF EXISTS expires_at,
  DROP COLUMN IF EXISTS purged_at;

DROP TRIGGER IF EXISTS trg_users_default_quota ON users;
DROP FUNCTION IF EXISTS create_default_quota();
DROP TABLE IF EXISTS user_quotas;
