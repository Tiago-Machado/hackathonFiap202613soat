-- 000002_quotas_retention_audit.up.sql
-- Cotas por usuário, retenção de dados e trilha de auditoria (LGPD).

-- Cotas por usuário ----------------------------------------------------------
-- Camada de banco (limites de negócio). O rate limit por minuto fica no Redis.
CREATE TABLE user_quotas (
  user_id             UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  max_uploads_per_day INT    NOT NULL DEFAULT 50,
  max_storage_bytes   BIGINT NOT NULL DEFAULT 5368709120,  -- 5 GiB
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_user_quotas_updated_at
  BEFORE UPDATE ON user_quotas
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Toda criação de usuário recebe uma cota padrão automaticamente.
CREATE OR REPLACE FUNCTION create_default_quota()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO user_quotas (user_id) VALUES (NEW.id);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_default_quota
  AFTER INSERT ON users
  FOR EACH ROW EXECUTE FUNCTION create_default_quota();

-- Retenção (LGPD: minimização) -----------------------------------------------
-- expires_at: quando o vídeo/zip deve ser purgado. purged_at: quando foi.
ALTER TABLE videos
  ADD COLUMN expires_at TIMESTAMPTZ,
  ADD COLUMN purged_at  TIMESTAMPTZ;

-- Índice parcial: o worker de retenção só varre o que ainda precisa ser purgado.
CREATE INDEX idx_videos_expiry ON videos (expires_at)
  WHERE purged_at IS NULL AND expires_at IS NOT NULL;

-- Auditoria (LGPD: accountability — quem fez o quê, quando) -------------------
CREATE TABLE audit_events (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID        REFERENCES users(id) ON DELETE SET NULL,
  action     TEXT        NOT NULL,   -- 'video.upload', 'video.download', 'user.erasure'...
  resource   TEXT,                   -- normalmente videos.id
  ip_address INET,
  metadata   JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_user_created ON audit_events (user_id, created_at DESC);
