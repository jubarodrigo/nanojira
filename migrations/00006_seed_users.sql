-- +goose Up
INSERT INTO users (id, name, email, role) VALUES
    ('11111111-1111-1111-1111-111111111101', 'Alice Manager', 'alice.manager@example.com', 'manager'),
    ('11111111-1111-1111-1111-111111111102', 'Bob Manager', 'bob.manager@example.com', 'manager'),
    ('22222222-2222-2222-2222-222222222201', 'Carol Worker', 'carol.worker@example.com', 'worker'),
    ('22222222-2222-2222-2222-222222222202', 'Dave Worker', 'dave.worker@example.com', 'worker'),
    ('22222222-2222-2222-2222-222222222203', 'Eve Worker', 'eve.worker@example.com', 'worker');

-- +goose Down
DELETE FROM users WHERE id IN (
    '11111111-1111-1111-1111-111111111101',
    '11111111-1111-1111-1111-111111111102',
    '22222222-2222-2222-2222-222222222201',
    '22222222-2222-2222-2222-222222222202',
    '22222222-2222-2222-2222-222222222203'
);
