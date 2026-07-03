-- +goose Up
CREATE TABLE assignment_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id),
    assignee_id UUID NOT NULL REFERENCES users(id),
    email TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assignment_notifications_task_id ON assignment_notifications(task_id);

-- +goose Down
DROP TABLE IF EXISTS assignment_notifications;
