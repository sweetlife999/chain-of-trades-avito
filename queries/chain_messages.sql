-- Доступ к треду: существование обмена, участие пользователя и статус — за один запрос.
-- Полный JOIN из GetExchangeByID ради проверки прав дёргать незачем.
-- name: GetExchangeAccess :one
SELECT
    exchange.status,
    EXISTS (
        SELECT 1
        FROM chain_participants AS participant
        WHERE participant.chain_id = exchange.id
          AND participant.user_id = sqlc.arg(user_id)
    ) AS is_participant
FROM chains AS exchange
WHERE exchange.id = sqlc.arg(exchange_id);

-- Отправитель получает назад ровно ту же форму сообщения, что придёт следующим
-- чтением треда, поэтому RETURNING идёт через CTE с автором. LEFT JOIN здесь не ради
-- NULL-автора (у текста он обязателен), а чтобы тип строки совпал с ListChainMessages.
-- name: CreateChainMessage :one
WITH inserted AS (
    INSERT INTO chain_messages (chain_id, author_id, body)
    VALUES (sqlc.arg(exchange_id), sqlc.arg(author_id), sqlc.arg(body))
    RETURNING id, kind, body, created_at, author_id
)
SELECT
    inserted.id,
    inserted.kind,
    inserted.body,
    inserted.created_at,
    inserted.author_id,
    author.nickname  AS author_nickname,
    author.photo_url AS author_photo_url
FROM inserted
LEFT JOIN users AS author
    ON author.id = inserted.author_id;

-- Событие сделки. author_id остаётся NULL, если событие принадлежит всему обмену,
-- а не конкретному участнику. body у событий пустой — текст собирает frontend.
-- name: CreateChainSystemMessage :exec
INSERT INTO chain_messages (chain_id, author_id, kind)
VALUES (sqlc.arg(exchange_id), sqlc.arg(author_id), sqlc.arg(kind));

-- LEFT JOIN: у событий обмена автора нет.
-- name: ListChainMessages :many
SELECT
    message.id,
    message.kind,
    message.body,
    message.created_at,
    message.author_id,
    author.nickname  AS author_nickname,
    author.photo_url AS author_photo_url
FROM chain_messages AS message
LEFT JOIN users AS author
    ON author.id = message.author_id
WHERE message.chain_id = sqlc.arg(exchange_id)
ORDER BY message.created_at, message.id;

-- Отметка ставится по последнему сообщению, которое клиент реально показал: время берётся
-- из самой строки треда, а не с часов сервера. Поэтому сообщение, попавшее в базу уже после
-- ответа на чтение, пометить прочитанным невозможно, и обе метки по построению лежат в одной
-- шкале. greatest() делает вызов монотонным и идемпотентным: повтор, две открытые вкладки и
-- id из чужого треда ничего не сдвигают назад, потому что greatest игнорирует NULL.
-- name: MarkChainMessagesRead :exec
UPDATE chain_participants
SET messages_read_at = greatest(
        messages_read_at,
        (
            SELECT message.created_at
            FROM chain_messages AS message
            WHERE message.id = sqlc.arg(last_message_id)
              AND message.chain_id = sqlc.arg(exchange_id)
        )
    )
WHERE chain_id = sqlc.arg(exchange_id)
  AND user_id = sqlc.arg(user_id);
