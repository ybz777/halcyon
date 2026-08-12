CREATE TABLE notifications (
    id               TEXT PRIMARY KEY,
    organization_id  TEXT NOT NULL,
    user_id          TEXT NOT NULL,
    event_id         TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    notification_type TEXT NOT NULL,
    title            TEXT NOT NULL,
    body             TEXT NOT NULL DEFAULT '',
    resource_type    TEXT NOT NULL DEFAULT '',
    resource_id      TEXT NOT NULL DEFAULT '',
    metadata         JSONB NOT NULL DEFAULT '{}',
    read_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, user_id, notification_type)
);

CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_org_created ON notifications (organization_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_user_unread ON notifications (user_id, read_at, created_at DESC) WHERE read_at IS NULL;

CREATE TABLE notification_preferences (
    user_id           TEXT NOT NULL,
    notification_type TEXT NOT NULL,
    channel           TEXT NOT NULL CHECK (channel IN ('IN_APP', 'EMAIL')),
    enabled           BOOLEAN NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, notification_type, channel)
);

CREATE INDEX idx_notification_preferences_user ON notification_preferences (user_id);

CREATE TABLE email_deliveries (
    id                   TEXT PRIMARY KEY,
    notification_id      TEXT NOT NULL REFERENCES notifications(id),
    user_id              TEXT NOT NULL,
    email                TEXT NOT NULL,
    template             TEXT NOT NULL,
    status               TEXT NOT NULL CHECK (status IN ('PENDING', 'DELIVERING', 'SENT', 'FAILED', 'RETRY_WAIT')),
    attempt_count        INT NOT NULL DEFAULT 0,
    available_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_attempt_at      TIMESTAMPTZ,
    next_attempt_at      TIMESTAMPTZ,
    provider             TEXT NOT NULL DEFAULT '',
    provider_message_id  TEXT NOT NULL DEFAULT '',
    last_error           TEXT NOT NULL DEFAULT '',
    locked_at            TIMESTAMPTZ,
    locked_by            TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at         TIMESTAMPTZ
);

CREATE INDEX idx_email_deliveries_ready ON email_deliveries (status, available_at, next_attempt_at);
CREATE INDEX idx_email_deliveries_notification ON email_deliveries (notification_id);
