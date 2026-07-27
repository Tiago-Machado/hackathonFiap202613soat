-- 000001_init.up.sql
-- Schema base: usuários, vídeos (jobs), outbox e refresh tokens.
-- PostgreSQL 13+.

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- users -----------------------------------------------------------------------
CREATE TABLE users (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email         CITEXT      NOT NULL UNIQUE,
  password_hash TEXT        NOT NULL,
  is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- videos ----------------------------------------------------------------------
CREATE TABLE videos (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  original_filename TEXT        NOT NULL,
  storage_key       TEXT        NOT NULL,
  content_type      TEXT,
  size_bytes        BIGINT,
  status            TEXT        NOT NULL DEFAULT 'PENDING'
                      CHECK (status IN ('PENDING','PROCESSING','DONE','ERROR')),
  attempts          INT         NOT NULL DEFAULT 0,
  error_message     TEXT,
  output_key        TEXT,
  output_size_bytes BIGINT,
  frame_count       INT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at        TIMESTAMPTZ,
  finished_at       TIMESTAMPTZ,
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_videos_updated_at
  BEFORE UPDATE ON videos
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_videos_user_created ON videos (user_id, created_at DESC);
CREATE INDEX idx_videos_status       ON videos (status);

-- outbox (Transactional Outbox Pattern) ---------------------------------------
CREATE TABLE outbox (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  aggregate_type TEXT        NOT NULL,
  aggregate_id   UUID        NOT NULL,
  event_type     TEXT        NOT NULL,
  payload        JSONB       NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at   TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at)
  WHERE published_at IS NULL;

-- refresh_tokens --------------------------------------------------------------
CREATE TABLE refresh_tokens (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT        NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
