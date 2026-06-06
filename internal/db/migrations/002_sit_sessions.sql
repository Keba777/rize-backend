CREATE TABLE IF NOT EXISTS sit_sessions (
    id           BIGSERIAL PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at   TIMESTAMPTZ NOT NULL,
    ended_at     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON sit_sessions (user_id, started_at DESC);
CREATE INDEX ON sit_sessions (user_id, ended_at) WHERE ended_at IS NULL;
