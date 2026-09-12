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

-- ---------------------------------------------------------------------------
-- Campaigns (existing + seed extras)
-- ---------------------------------------------------------------------------
-- INSERT INTO campaigns (
--     id, client_id, name, is_default, background_status,
--     start_at, end_at, created_at, updated_at
-- ) VALUES
--     -- personal client
--     ('c6b095d1-d177-48de-8a78-017c6baf62d0', '1ee171fe-8424-41ec-819d-06ab615de7fc', 'Default', TRUE, 'none',
--      NULL, NULL, '2026-09-10 18:20:19+00', '2026-09-10 18:20:19+00'),
--     ('ab1a6078-2b4f-4b16-9d98-a132c2724e59', '1ee171fe-8424-41ec-819d-06ab615de7fc', 'Chaussea', FALSE, 'ready',
--      '2026-09-11 22:00:00+00', '2026-09-29 22:00:00+00', '2026-09-11 10:00:00+00', '2026-09-11 10:00:00+00'),
--     ('fa530336-437d-44c7-bcaf-cde78a25d083', '1ee171fe-8424-41ec-819d-06ab615de7fc', 'Campagne numero 1', FALSE, 'none',
--      '2026-09-09 22:00:00+00', '2026-09-29 22:00:00+00', '2026-09-10 12:00:00+00', '2026-09-10 12:00:00+00'),
--     ('a1000001-0001-4000-8000-000000000001', '1ee171fe-8424-41ec-819d-06ab615de7fc', 'Summer Launch', FALSE, 'none',
--      '2026-04-01 00:00:00+00', '2026-06-30 23:59:59+00', '2026-03-15 10:00:00+00', '2026-03-15 10:00:00+00'),
--     ('a1000001-0001-4000-8000-000000000002', '1ee171fe-8424-41ec-819d-06ab615de7fc', 'Black Friday', FALSE, 'pending',
--      '2026-11-01 00:00:00+00', '2026-11-30 23:59:59+00', '2026-08-01 10:00:00+00', '2026-08-01 10:00:00+00'),
--     ('a1000001-0001-4000-8000-000000000003', '1ee171fe-8424-41ec-819d-06ab615de7fc', 'Product Shoot', FALSE, 'ready',
--      '2026-07-01 00:00:00+00', '2026-09-15 23:59:59+00', '2026-06-20 10:00:00+00', '2026-06-20 10:00:00+00'),
--     ('a1000001-0001-4000-8000-000000000004', '1ee171fe-8424-41ec-819d-06ab615de7fc', 'Influencer Wave', FALSE, 'failed',
--      '2026-05-01 00:00:00+00', '2026-05-31 23:59:59+00', '2026-04-25 10:00:00+00', '2026-04-25 10:00:00+00'),
--     -- chaussea client
--     ('9c7c1bdf-f6b9-471e-8e70-aee02cd36c63', 'c4fc07fc-1fd3-4e70-a794-96ca2d4eaeef', 'Default', TRUE, 'none',
--      NULL, NULL, '2026-09-10 18:20:28+00', '2026-09-10 18:20:28+00'),
--     ('a1000002-0002-4000-8000-000000000001', 'c4fc07fc-1fd3-4e70-a794-96ca2d4eaeef', 'Retail Spring', FALSE, 'ready',
--      '2026-03-01 00:00:00+00', '2026-05-31 23:59:59+00', '2026-02-20 10:00:00+00', '2026-02-20 10:00:00+00'),
--     ('a1000002-0002-4000-8000-000000000002', 'c4fc07fc-1fd3-4e70-a794-96ca2d4eaeef', 'Retail Autumn', FALSE, 'none',
--      '2026-09-01 00:00:00+00', '2026-11-30 23:59:59+00', '2026-08-15 10:00:00+00', '2026-08-15 10:00:00+00')
-- ON CONFLICT (id) DO UPDATE SET
--     name = EXCLUDED.name,
--     is_default = EXCLUDED.is_default,
--     background_status = EXCLUDED.background_status,
--     start_at = EXCLUDED.start_at,
--     end_at = EXCLUDED.end_at,
--     updated_at = EXCLUDED.updated_at;

-- -- Replace previous seed contents only (safe for re-run).
-- DELETE FROM content_analysis_results
-- WHERE content_id IN (SELECT id FROM contents WHERE object_key LIKE 'seed/%');
-- DELETE FROM contents WHERE object_key LIKE 'seed/%';

-- -- ---------------------------------------------------------------------------
-- -- Contents: all statuses + labels, spread over the last 6 months
-- -- ---------------------------------------------------------------------------
-- DO $$
-- DECLARE
--     v_client_personal UUID := '1ee171fe-8424-41ec-819d-06ab615de7fc';
--     v_client_chaussea UUID := 'c4fc07fc-1fd3-4e70-a794-96ca2d4eaeef';
--     v_campaigns UUID[] := ARRAY[
--         'c6b095d1-d177-48de-8a78-017c6baf62d0'::uuid,
--         'ab1a6078-2b4f-4b16-9d98-a132c2724e59'::uuid,
--         'fa530336-437d-44c7-bcaf-cde78a25d083'::uuid,
--         'a1000001-0001-4000-8000-000000000001'::uuid,
--         'a1000001-0001-4000-8000-000000000002'::uuid,
--         'a1000001-0001-4000-8000-000000000003'::uuid,
--         'a1000001-0001-4000-8000-000000000004'::uuid,
--         '9c7c1bdf-f6b9-471e-8e70-aee02cd36c63'::uuid,
--         'a1000002-0002-4000-8000-000000000001'::uuid,
--         'a1000002-0002-4000-8000-000000000002'::uuid
--     ];
--     v_clients UUID[] := ARRAY[
--         v_client_personal, v_client_personal, v_client_personal, v_client_personal,
--         v_client_personal, v_client_personal, v_client_personal,
--         v_client_chaussea, v_client_chaussea, v_client_chaussea
--     ];
--     v_statuses TEXT[] := ARRAY['pending_upload', 'uploaded', 'analyzing', 'analyzed', 'failed'];
--     v_labels TEXT[] := ARRAY['human', 'ai_generated', 'uncertain'];
--     v_month_start TIMESTAMPTZ;
--     v_created TIMESTAMPTZ;
--     v_content_id UUID;
--     v_campaign_id UUID;
--     v_client_id UUID;
--     v_status TEXT;
--     v_label TEXT;
--     v_confidence DOUBLE PRECISION;
--     v_filename TEXT;
--     v_object_key TEXT;
--     v_thumb_key TEXT;
--     v_size BIGINT;
--     v_i INT;
--     v_m INT;
--     v_s INT;
--     v_l INT;
--     v_n INT;
--     v_seq INT := 0;
-- BEGIN
--     -- 6 months window ending at current UTC month.
--     FOR v_m IN 0..5 LOOP
--         v_month_start := date_trunc('month', NOW() AT TIME ZONE 'UTC') - (v_m || ' months')::interval;

--         -- Per month: every status at least once, analyzed × each label × several.
--         FOR v_s IN 1..array_length(v_statuses, 1) LOOP
--             v_status := v_statuses[v_s];

--             IF v_status = 'analyzed' THEN
--                 FOR v_l IN 1..array_length(v_labels, 1) LOOP
--                     v_label := v_labels[v_l];
--                     v_confidence := CASE v_label
--                         WHEN 'human' THEN 0.85 + (v_m % 3) * 0.04
--                         WHEN 'ai_generated' THEN 0.78 + (v_m % 4) * 0.05
--                         ELSE 0.45 + (v_m % 2) * 0.05
--                     END;

--                     -- 4 analyzed contents per label per month
--                     FOR v_n IN 1..4 LOOP
--                         v_seq := v_seq + 1;
--                         v_i := ((v_seq - 1) % array_length(v_campaigns, 1)) + 1;
--                         v_campaign_id := v_campaigns[v_i];
--                         v_client_id := v_clients[v_i];
--                         v_content_id := gen_random_uuid();
--                         v_created := v_month_start
--                             + ((v_n + v_l * 2 + v_m) % 25 || ' days')::interval
--                             + ((8 + v_n) || ' hours')::interval
--                             + ((v_seq % 50) || ' minutes')::interval;
--                         v_filename := format('seed-%s-%s-%s.jpg', v_status, v_label, v_seq);
--                         v_object_key := format(
--                             'seed/clients/%s/campaigns/%s/contents/%s.jpg',
--                             v_client_id, v_campaign_id, v_content_id
--                         );
--                         v_thumb_key := format(
--                             'seed/clients/%s/campaigns/%s/contents/thumbnails/%s.jpg',
--                             v_client_id, v_campaign_id, v_content_id
--                         );
--                         v_size := 20000 + (v_seq * 137) % 900000;

--                         INSERT INTO contents (
--                             id, campaign_id, client_id, filename, content_type,
--                             object_key, thumbnail_key, size_bytes, status, label, confidence,
--                             created_at, updated_at
--                         ) VALUES (
--                             v_content_id, v_campaign_id, v_client_id, v_filename, 'image/jpeg',
--                             v_object_key, v_thumb_key, v_size, v_status, v_label, v_confidence,
--                             v_created, v_created + interval '30 minutes'
--                         );

--                         INSERT INTO content_analysis_results (
--                             id, content_id, detector_name, status, signals, error, started_at, completed_at
--                         ) VALUES
--                             (
--                                 gen_random_uuid(), v_content_id, 'metadata', 'success',
--                                 jsonb_build_array(jsonb_build_object(
--                                     'type', 'meta', 'code', 'object_present',
--                                     'description', 'Seed metadata signal', 'weight', 0.1
--                                 )),
--                                 NULL, v_created, v_created + interval '2 seconds'
--                             ),
--                             (
--                                 gen_random_uuid(), v_content_id, 'heuristic', 'success',
--                                 jsonb_build_array(jsonb_build_object(
--                                     'type', 'heuristic', 'code', 'format_ok',
--                                     'description', 'Seed heuristic signal', 'weight', 0.05
--                                 )),
--                                 NULL, v_created, v_created + interval '3 seconds'
--                             ),
--                             (
--                                 gen_random_uuid(), v_content_id, 'sightengine', 'success',
--                                 jsonb_build_array(jsonb_build_object(
--                                     'type', 'model', 'code', 'sightengine_genai',
--                                     'description', format('Seed Sightengine %s', v_label),
--                                     'weight', v_confidence
--                                 )),
--                                 NULL, v_created, v_created + interval '8 seconds'
--                             );
--                     END LOOP;
--                 END LOOP;
--             ELSE
--                 -- 3 contents per non-analyzed status per month
--                 FOR v_n IN 1..3 LOOP
--                     v_seq := v_seq + 1;
--                     v_i := ((v_seq - 1) % array_length(v_campaigns, 1)) + 1;
--                     v_campaign_id := v_campaigns[v_i];
--                     v_client_id := v_clients[v_i];
--                     v_content_id := gen_random_uuid();
--                     v_created := v_month_start
--                         + ((v_n + v_s * 3 + v_m) % 27 || ' days')::interval
--                         + ((10 + v_n) || ' hours')::interval;
--                     v_filename := format('seed-%s-%s.jpg', v_status, v_seq);
--                     v_object_key := format(
--                         'seed/clients/%s/campaigns/%s/contents/%s.jpg',
--                         v_client_id, v_campaign_id, v_content_id
--                     );
--                     v_thumb_key := CASE
--                         WHEN v_status = 'pending_upload' THEN NULL
--                         ELSE format(
--                             'seed/clients/%s/campaigns/%s/contents/thumbnails/%s.jpg',
--                             v_client_id, v_campaign_id, v_content_id
--                         )
--                     END;
--                     v_size := CASE
--                         WHEN v_status = 'pending_upload' THEN NULL
--                         ELSE 15000 + (v_seq * 211) % 800000
--                     END;

--                     INSERT INTO contents (
--                         id, campaign_id, client_id, filename, content_type,
--                         object_key, thumbnail_key, size_bytes, status, label, confidence,
--                         created_at, updated_at
--                     ) VALUES (
--                         v_content_id, v_campaign_id, v_client_id, v_filename, 'image/jpeg',
--                         v_object_key, v_thumb_key, v_size, v_status, NULL, NULL,
--                         v_created, v_created
--                     );
--                 END LOOP;
--             END IF;
--         END LOOP;
--     END LOOP;
-- END $$;

-- -- +goose StatementEnd

-- -- +goose Down
-- -- +goose StatementBegin

-- DELETE FROM content_analysis_results
-- WHERE content_id IN (SELECT id FROM contents WHERE object_key LIKE 'seed/%');

-- DELETE FROM contents WHERE object_key LIKE 'seed/%';

-- DELETE FROM campaigns WHERE id IN (
--     'a1000001-0001-4000-8000-000000000001',
--     'a1000001-0001-4000-8000-000000000002',
--     'a1000001-0001-4000-8000-000000000003',
--     'a1000001-0001-4000-8000-000000000004',
--     'a1000002-0002-4000-8000-000000000001',
--     'a1000002-0002-4000-8000-000000000002'
-- );

-- -- Keep identity (user/clients/default campaigns) on down — only remove seed extras above.

-- -- +goose StatementEnd
