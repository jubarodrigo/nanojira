-- +goose Up
CREATE TABLE stepback_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id),
    requested_by_id UUID NOT NULL REFERENCES users(id),
    from_status TEXT NOT NULL,
    to_status TEXT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by_id UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stepback_requests_task_id ON stepback_requests(task_id);
CREATE INDEX idx_stepback_requests_status ON stepback_requests(status);

-- +goose Down
DROP TABLE IF EXISTS stepback_requests;
