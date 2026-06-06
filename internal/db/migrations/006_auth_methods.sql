ALTER TABLE users
  ADD COLUMN IF NOT EXISTS password_hash  TEXT,
  ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS google_id      TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS users_google_id_idx
  ON users (google_id) WHERE google_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS email_verify_tokens (
  id         BIGSERIAL   PRIMARY KEY,
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token      TEXT        UNIQUE NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used       BOOLEAN     NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS email_verify_tokens_token_idx ON email_verify_tokens (token);

-- Existing magic-link users are already email-verified
UPDATE users SET email_verified = true;
