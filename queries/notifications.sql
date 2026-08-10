-- name: ListNotifications :many
SELECT
    notification.id,
    notification.chain_id,
    notification.support_thread_id,
    COALESCE(notification.message_id, notification.support_message_id) AS message_id,
    notification.kind,
    COALESCE(message.author_id, support_message.author_id) AS author_id,
    author.nickname AS author_nickname,
    author.photo_url AS author_photo_url,
    COALESCE(exchange.status::text, '') AS exchange_status,
    gives_item.title AS gives_item_title,
    receives_item.title AS receives_item_title,
    support_thread.subject AS support_subject,
    notification.read_at,
    notification.created_at
FROM notifications AS notification
LEFT JOIN chains AS exchange
    ON exchange.id = notification.chain_id
LEFT JOIN chain_participants AS current_participant
    ON current_participant.chain_id = notification.chain_id
   AND current_participant.user_id = notification.user_id
LEFT JOIN items AS gives_item
    ON gives_item.id = current_participant.gives_item_id
LEFT JOIN items AS receives_item
    ON receives_item.id = current_participant.receives_item_id
LEFT JOIN chain_messages AS message
    ON message.id = notification.message_id
LEFT JOIN support_threads AS support_thread
    ON support_thread.id = notification.support_thread_id
LEFT JOIN support_messages AS support_message
    ON support_message.id = notification.support_message_id
LEFT JOIN users AS author
    ON author.id = COALESCE(message.author_id, support_message.author_id)
WHERE notification.user_id = sqlc.arg(user_id)
  AND (NOT sqlc.arg(unread_only)::boolean OR notification.read_at IS NULL)
ORDER BY notification.created_at DESC, notification.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountUnreadNotifications :one
SELECT count(*)
FROM notifications
WHERE user_id = sqlc.arg(user_id)
  AND read_at IS NULL;

-- name: MarkNotificationRead :execrows
UPDATE notifications
SET read_at = COALESCE(read_at, clock_timestamp())
WHERE id = sqlc.arg(notification_id)
  AND user_id = sqlc.arg(user_id);

-- name: MarkAllNotificationsRead :execrows
UPDATE notifications
SET read_at = clock_timestamp()
WHERE user_id = sqlc.arg(user_id)
  AND read_at IS NULL;
