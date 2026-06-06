CREATE TABLE IF NOT EXISTS daily_reports (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    report_date         DATE NOT NULL,
    total_sit_min       INT NOT NULL DEFAULT 0,
    total_move_min      INT NOT NULL DEFAULT 0,
    longest_sit_min     INT NOT NULL DEFAULT 0,
    sit_sessions_count  INT NOT NULL DEFAULT 0,
    move_sessions_count INT NOT NULL DEFAULT 0,
    health_score        INT NOT NULL DEFAULT 0,
    advice              TEXT,
    generated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, report_date)
);
CREATE INDEX ON daily_reports (user_id, report_date DESC);
