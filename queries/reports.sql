-- Цель жалобы и право на неё — за один запрос, как GetExchangeAccess перед записью
-- в тред. Вставка с WHERE вернула бы ноль строк без причины, а причин отказа три и у
-- каждой свой код ответа: посторонний, собственное сообщение и событие обмена.
-- name: GetReportTarget :one
SELECT
    message.kind,
    message.author_id,
    EXISTS (
        SELECT 1
        FROM chain_participants AS participant
        WHERE participant.chain_id = message.chain_id
          AND participant.user_id = sqlc.arg(user_id)
    ) AS is_participant
FROM chain_messages AS message
WHERE message.id = sqlc.arg(message_id);

-- Повтор ловится UNIQUE (reporter_id, message_id), а не проверкой перед вставкой:
-- две открытые вкладки не создадут двух жалоб.
-- name: CreateReport :one
INSERT INTO reports (reporter_id, message_id, reason, comment)
VALUES (
    sqlc.arg(reporter_id),
    sqlc.arg(message_id),
    sqlc.arg(reason),
    sqlc.arg(comment)
)
RETURNING *;

-- Административная очередь. Пустая строка означает, что фильтр не задан; UUID
-- разбирается в handler и приходит сюда NULL, если фильтра по исполнителю нет.
-- Старые жалобы идут первыми: очередь модерации разбирается по порядку поступления.
-- name: ListReportsForAdmin :many
SELECT
    report.id         AS report_id,
    report.reason     AS report_reason,
    report.comment    AS report_comment,
    report.status     AS report_status,
    report.created_at AS report_created_at,
    report.assigned_at AS report_assigned_at,
    report.closed_at AS report_closed_at,
    report.resolution_comment AS report_resolution_comment,
    reporter.id        AS reporter_id,
    reporter.nickname  AS reporter_nickname,
    reporter.photo_url AS reporter_photo_url,
    offender.id        AS offender_id,
    offender.nickname  AS offender_nickname,
    offender.photo_url AS offender_photo_url,
    message.id         AS message_id,
    message.body       AS message_body,
    message.created_at AS message_created_at,
    exchange.id        AS exchange_id,
    exchange.status    AS exchange_status,
    assignee.id        AS assignee_id,
    assignee.nickname  AS assignee_nickname,
    assignee.photo_url AS assignee_photo_url
FROM reports AS report
JOIN users AS reporter
    ON reporter.id = report.reporter_id
JOIN chain_messages AS message
    ON message.id = report.message_id
JOIN users AS offender
    ON offender.id = message.author_id
JOIN chains AS exchange
    ON exchange.id = message.chain_id
LEFT JOIN users AS assignee
    ON assignee.id = report.assignee_id
WHERE (sqlc.arg(status)::text = '' OR report.status::text = sqlc.arg(status)::text)
  AND (sqlc.arg(reason)::text = '' OR report.reason::text = sqlc.arg(reason)::text)
  AND (sqlc.narg(assignee_id)::uuid IS NULL OR report.assignee_id = sqlc.narg(assignee_id)::uuid)
ORDER BY report.created_at, report.id
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountReportsForAdmin :one
SELECT count(*)
FROM reports AS report
WHERE (sqlc.arg(status)::text = '' OR report.status::text = sqlc.arg(status)::text)
  AND (sqlc.arg(reason)::text = '' OR report.reason::text = sqlc.arg(reason)::text)
  AND (sqlc.narg(assignee_id)::uuid IS NULL OR report.assignee_id = sqlc.narg(assignee_id)::uuid);

-- Карточка использует ту же форму, что и строка очереди. Так frontend может перейти
-- от списка к деталям без второго набора несовместимых названий полей.
-- name: GetReportForAdmin :one
SELECT
    report.id         AS report_id,
    report.reason     AS report_reason,
    report.comment    AS report_comment,
    report.status     AS report_status,
    report.created_at AS report_created_at,
    report.assigned_at AS report_assigned_at,
    report.closed_at AS report_closed_at,
    report.resolution_comment AS report_resolution_comment,
    reporter.id        AS reporter_id,
    reporter.nickname  AS reporter_nickname,
    reporter.photo_url AS reporter_photo_url,
    offender.id        AS offender_id,
    offender.nickname  AS offender_nickname,
    offender.photo_url AS offender_photo_url,
    message.id         AS message_id,
    message.body       AS message_body,
    message.created_at AS message_created_at,
    exchange.id        AS exchange_id,
    exchange.status    AS exchange_status,
    assignee.id        AS assignee_id,
    assignee.nickname  AS assignee_nickname,
    assignee.photo_url AS assignee_photo_url
FROM reports AS report
JOIN users AS reporter
    ON reporter.id = report.reporter_id
JOIN chain_messages AS message
    ON message.id = report.message_id
JOIN users AS offender
    ON offender.id = message.author_id
JOIN chains AS exchange
    ON exchange.id = message.chain_id
LEFT JOIN users AS assignee
    ON assignee.id = report.assignee_id
WHERE report.id = sqlc.arg(report_id);

-- Назначение — один условный UPDATE. Два администратора могут нажать кнопку одновременно,
-- но строку изменит только один: после его UPDATE assignee_id уже не NULL.
-- name: AssignReportForAdmin :execrows
WITH changed AS (
    UPDATE reports AS report
    SET
        assignee_id = sqlc.arg(admin_id),
        assigned_at = now()
    WHERE report.id = sqlc.arg(report_id)
      AND report.status = 'open'
      AND report.assignee_id IS NULL
    RETURNING report.id
)
INSERT INTO admin_audit_log (admin_id, action, target_type, target_id)
SELECT sqlc.arg(admin_id), 'report_assigned', 'report', changed.id
FROM changed;

-- Решение может принять только текущий исполнитель и только пока жалоба open.
-- status и время закрытия меняются одним атомарным запросом.
-- name: DecideReportForAdmin :execrows
WITH changed AS (
    UPDATE reports AS report
    SET
        status = sqlc.arg(decision),
        closed_at = now(),
        resolution_comment = sqlc.arg(resolution_comment)
    WHERE report.id = sqlc.arg(report_id)
      AND report.status = 'open'
      AND report.assignee_id = sqlc.arg(admin_id)
    RETURNING report.id
)
INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, metadata)
SELECT
    sqlc.arg(admin_id),
    CASE
        WHEN sqlc.arg(decision)::text = 'resolved' THEN 'report_resolved'::admin_audit_action
        ELSE 'report_rejected'::admin_audit_action
    END,
    'report',
    changed.id,
    jsonb_build_object('resolution_comment', sqlc.arg(resolution_comment)::text)
FROM changed;

-- Вызывается только если условный UPDATE не изменил строку. Нужен, чтобы отличить 404
-- от конфликтов «уже назначена», «не назначена» и «уже закрыта».
-- name: GetReportProcessingState :one
SELECT status, assignee_id
FROM reports AS report
WHERE report.id = sqlc.arg(report_id);
