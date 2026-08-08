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
