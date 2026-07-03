-- +goose Up
INSERT INTO tasks (id, title, description, status, reporter_id, assignee_id) VALUES
    ('33333333-3333-3333-3333-333333333301', 'Setup staging environment', 'Provision and configure staging servers', 'todo', '11111111-1111-1111-1111-111111111101', NULL),
    ('33333333-3333-3333-3333-333333333302', 'Fix login timeout bug', 'Users are logged out after 5 minutes instead of 30', 'doing', '11111111-1111-1111-1111-111111111101', '22222222-2222-2222-2222-222222222201'),
    ('33333333-3333-3333-3333-333333333303', 'Write API documentation', 'Document all public endpoints for integrators', 'testing', '11111111-1111-1111-1111-111111111102', '22222222-2222-2222-2222-222222222202'),
    ('33333333-3333-3333-3333-333333333304', 'Migrate legacy reports', 'Move monthly reports from spreadsheets to the new system', 'on_hold', '11111111-1111-1111-1111-111111111102', '22222222-2222-2222-2222-222222222203'),
    ('33333333-3333-3333-3333-333333333305', 'Deploy v1.0 release', 'Coordinate production deployment checklist', 'done', '11111111-1111-1111-1111-111111111101', '22222222-2222-2222-2222-222222222201');

-- +goose Down
DELETE FROM tasks WHERE id IN (
    '33333333-3333-3333-3333-333333333301',
    '33333333-3333-3333-3333-333333333302',
    '33333333-3333-3333-3333-333333333303',
    '33333333-3333-3333-3333-333333333304',
    '33333333-3333-3333-3333-333333333305'
);
