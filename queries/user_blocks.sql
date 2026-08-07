-- Блокировка и отмена ещё не подтверждённых совместных обменов выполняются одним
-- SQL-выражением, то есть атомарно. Подтверждённые и завершённые обмены не меняются.
-- name: AddUserBlockAndCancelProposals :execrows
WITH inserted_block AS (
    INSERT INTO user_blocks (blocker_id, blocked_id)
    VALUES (sqlc.arg(blocker_id), sqlc.arg(blocked_id))
    ON CONFLICT (blocker_id, blocked_id) DO NOTHING
    RETURNING blocker_id
)
UPDATE chains AS exchange
SET status = 'cancelled',
    closed_at = now()
WHERE exchange.status = 'proposed'
  AND EXISTS (
      SELECT 1
      FROM chain_participants AS participant
      WHERE participant.chain_id = exchange.id
        AND participant.user_id = sqlc.arg(blocker_id)
  )
  AND EXISTS (
      SELECT 1
      FROM chain_participants AS participant
      WHERE participant.chain_id = exchange.id
        AND participant.user_id = sqlc.arg(blocked_id)
  );

-- name: ListUserBlocks :many
SELECT
    blocked_user.id,
    blocked_user.nickname,
    blocked_user.photo_url,
    block.created_at AS blocked_at
FROM user_blocks AS block
JOIN users AS blocked_user
  ON blocked_user.id = block.blocked_id
WHERE block.blocker_id = sqlc.arg(blocker_id)
ORDER BY block.created_at DESC, blocked_user.id;

-- Повторное удаление безопасно: отсутствие строки не считается ошибкой.
-- name: DeleteUserBlock :exec
DELETE FROM user_blocks
WHERE blocker_id = sqlc.arg(blocker_id)
  AND blocked_id = sqlc.arg(blocked_id);
