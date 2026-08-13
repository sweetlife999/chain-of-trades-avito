-- name: ClaimAntiscamAnalysis :one
WITH candidate AS (
    SELECT analysis.id
    FROM antiscam_analyses AS analysis
    WHERE (
            analysis.status IN ('pending', 'failed')
            AND analysis.available_at <= now()
        )
       OR (
            analysis.status = 'processing'
            AND analysis.locked_at < now() - interval '5 minutes'
        )
    ORDER BY analysis.available_at, analysis.created_at, analysis.id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE antiscam_analyses AS analysis
SET status = 'processing', locked_at = now(), attempts = analysis.attempts + 1
FROM candidate
WHERE analysis.id = candidate.id
RETURNING analysis.id, analysis.message_id, analysis.attempts;

-- name: GetAntiscamInput :one
SELECT
    message.id AS message_id,
    message.chain_id,
    message.author_id,
    message.body,
    author.nickname AS author_nickname
FROM chain_messages AS message
JOIN users AS author ON author.id = message.author_id
WHERE message.id = sqlc.arg(message_id)
  AND message.kind = 'text';

-- Последние реплики до проверяемой включительно. Разворачивает их repository,
-- чтобы модель увидела диалог в обычном хронологическом порядке.
-- name: ListAntiscamContext :many
SELECT
    message.id,
    message.author_id,
    author.nickname,
    message.body,
    message.created_at
FROM chain_messages AS target
JOIN chain_messages AS message
    ON message.chain_id = target.chain_id
   AND message.kind = 'text'
   AND (message.created_at, message.id) <= (target.created_at, target.id)
JOIN users AS author ON author.id = message.author_id
WHERE target.id = sqlc.arg(message_id)
ORDER BY message.created_at DESC, message.id DESC
LIMIT 6;

-- Результат анализа и создание/обновление карточки выполняются атомарно.
-- name: CompleteAntiscamAnalysis :exec
WITH analyzed AS (
    UPDATE antiscam_analyses
    SET
        status = 'completed',
        locked_at = NULL,
        rule_score = sqlc.arg(rule_score),
        rule_hits = sqlc.arg(rule_hits),
        model_suspicious = sqlc.arg(model_suspicious),
        model_severity = sqlc.arg(model_severity),
        category = sqlc.narg(category),
        reason = sqlc.arg(reason),
        evidence = sqlc.arg(evidence),
        risk = sqlc.arg(risk),
        model_name = sqlc.arg(model_name),
        prompt_version = sqlc.arg(prompt_version),
        last_error = '',
        processed_at = now()
    WHERE antiscam_analyses.id = sqlc.arg(analysis_id)
    RETURNING antiscam_analyses.message_id
), target AS (
    SELECT message.id AS message_id, message.chain_id, message.author_id
    FROM analyzed
    JOIN chain_messages AS message ON message.id = analyzed.message_id
    WHERE sqlc.arg(is_suspicious)::boolean
), opened AS (
    INSERT INTO antiscam_cases (chain_id, suspect_user_id, risk, category, reason)
    SELECT target.chain_id, target.author_id, sqlc.arg(risk), sqlc.arg(category), sqlc.arg(reason)
    FROM target
    ON CONFLICT (chain_id, suspect_user_id) WHERE status = 'open'
    DO UPDATE SET
        risk = greatest(antiscam_cases.risk, excluded.risk),
        category = CASE
            WHEN excluded.risk >= antiscam_cases.risk THEN excluded.category
            ELSE antiscam_cases.category
        END,
        reason = CASE
            WHEN excluded.risk >= antiscam_cases.risk THEN excluded.reason
            ELSE antiscam_cases.reason
        END
    RETURNING id
)
INSERT INTO antiscam_case_messages (case_id, message_id)
SELECT opened.id, target.message_id
FROM opened CROSS JOIN target
ON CONFLICT DO NOTHING;

-- name: RetryAntiscamAnalysis :exec
UPDATE antiscam_analyses
SET
    status = 'failed',
    locked_at = NULL,
    available_at = now() + make_interval(secs => sqlc.arg(retry_seconds)::integer),
    last_error = left(sqlc.arg(last_error), 2000)
WHERE id = sqlc.arg(analysis_id);

-- name: ListAntiscamCasesForAdmin :many
SELECT
    antiscam_case.*,
    suspect.nickname AS suspect_nickname,
    suspect.photo_url AS suspect_photo_url,
    reviewer.nickname AS reviewer_nickname,
    evidence.message_id AS latest_message_id,
    evidence.body AS latest_message_body,
    evidence.created_at AS latest_message_created_at,
    evidence_count.total AS evidence_count
FROM antiscam_cases AS antiscam_case
JOIN users AS suspect ON suspect.id = antiscam_case.suspect_user_id
LEFT JOIN users AS reviewer ON reviewer.id = antiscam_case.reviewed_by
JOIN LATERAL (
    SELECT message.id AS message_id, message.body, message.created_at
    FROM antiscam_case_messages AS link
    JOIN chain_messages AS message ON message.id = link.message_id
    WHERE link.case_id = antiscam_case.id
    ORDER BY message.created_at DESC, message.id DESC
    LIMIT 1
) AS evidence ON true
JOIN LATERAL (
    SELECT count(*) AS total
    FROM antiscam_case_messages AS link
    WHERE link.case_id = antiscam_case.id
) AS evidence_count ON true
WHERE (sqlc.arg(case_status)::text = '' OR antiscam_case.status::text = sqlc.arg(case_status)::text)
  AND (sqlc.arg(case_category)::text = '' OR antiscam_case.category::text = sqlc.arg(case_category)::text)
  AND antiscam_case.risk >= sqlc.arg(min_risk)
ORDER BY antiscam_case.risk DESC, antiscam_case.updated_at DESC, antiscam_case.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountAntiscamCasesForAdmin :one
SELECT count(*)
FROM antiscam_cases AS antiscam_case
WHERE (sqlc.arg(case_status)::text = '' OR antiscam_case.status::text = sqlc.arg(case_status)::text)
  AND (sqlc.arg(case_category)::text = '' OR antiscam_case.category::text = sqlc.arg(case_category)::text)
  AND antiscam_case.risk >= sqlc.arg(min_risk);

-- name: GetAntiscamCaseForAdmin :one
SELECT
    antiscam_case.*,
    suspect.nickname AS suspect_nickname,
    suspect.photo_url AS suspect_photo_url,
    reviewer.nickname AS reviewer_nickname,
    evidence.message_id AS latest_message_id,
    evidence.body AS latest_message_body,
    evidence.created_at AS latest_message_created_at,
    evidence_count.total AS evidence_count
FROM antiscam_cases AS antiscam_case
JOIN users AS suspect ON suspect.id = antiscam_case.suspect_user_id
LEFT JOIN users AS reviewer ON reviewer.id = antiscam_case.reviewed_by
JOIN LATERAL (
    SELECT message.id AS message_id, message.body, message.created_at
    FROM antiscam_case_messages AS link
    JOIN chain_messages AS message ON message.id = link.message_id
    WHERE link.case_id = antiscam_case.id
    ORDER BY message.created_at DESC, message.id DESC
    LIMIT 1
) AS evidence ON true
JOIN LATERAL (
    SELECT count(*) AS total
    FROM antiscam_case_messages AS link
    WHERE link.case_id = antiscam_case.id
) AS evidence_count ON true
WHERE antiscam_case.id = sqlc.arg(case_id);

-- name: ListAntiscamEvidenceIDs :many
SELECT message_id
FROM antiscam_case_messages
WHERE case_id = sqlc.arg(case_id)
ORDER BY created_at, message_id;

-- name: DecideAntiscamCaseForAdmin :execrows
UPDATE antiscam_cases
SET
    status = CASE
        WHEN sqlc.arg(decision)::text = 'confirmed' THEN 'resolved'::antiscam_case_status
        ELSE 'dismissed'::antiscam_case_status
    END,
    reviewed_by = sqlc.arg(admin_id),
    decision = sqlc.arg(decision),
    resolution_comment = sqlc.arg(resolution_comment),
    closed_at = now()
WHERE id = sqlc.arg(case_id)
  AND status = 'open';
