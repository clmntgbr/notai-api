-- +goose Up
-- +goose StatementBegin

-- ---------------------------------------------------------------------------
-- Identity (same as current local/dev data)
-- ---------------------------------------------------------------------------
INSERT INTO clients (id, name, created_at, updated_at) VALUES
    ('1ee171fe-8424-41ec-819d-06ab615de7fc', 'Clément Goubier', '2026-09-10 18:20:19.805484+00', '2026-09-10 18:20:19.805492+00'),
    ('c4fc07fc-1fd3-4e70-a794-96ca2d4eaeef', 'Chaussea', '2026-09-10 18:20:28.510234+00', '2026-09-10 18:20:28.510268+00')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    updated_at = EXCLUDED.updated_at;

INSERT INTO users (
    id, clerk_id, first_name, last_name, banned, email,
    created_at, updated_at, current_client_id
) VALUES (
    '676404a5-7a8f-47da-9c3a-30141e6707e2',
    'user_3J9DW3DyMHhbpIXpSC2zKVeNdPT',
    'Clément',
    'Goubier',
    FALSE,
    'clement.goubier@gmail.com',
    '2026-09-10 18:20:19.798983+00',
    '2026-09-12 08:00:51.763065+00',
    '1ee171fe-8424-41ec-819d-06ab615de7fc'
)
ON CONFLICT (id) DO UPDATE SET
    clerk_id = EXCLUDED.clerk_id,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    banned = EXCLUDED.banned,
    email = EXCLUDED.email,
    current_client_id = EXCLUDED.current_client_id,
    updated_at = EXCLUDED.updated_at;

INSERT INTO user_clients (user_id, client_id, created_at) VALUES
    ('676404a5-7a8f-47da-9c3a-30141e6707e2', '1ee171fe-8424-41ec-819d-06ab615de7fc', '2026-09-10 18:20:19.809154+00'),
    ('676404a5-7a8f-47da-9c3a-30141e6707e2', 'c4fc07fc-1fd3-4e70-a794-96ca2d4eaeef', '2026-09-10 18:20:28.512284+00')
ON CONFLICT (user_id, client_id) DO NOTHING;
