-- name: CreateSupportThread :one
WITH created_thread AS (
    INSERT INTO support_threads (user_id, subject)
    VALUES (sqlc.arg(user_id), sqlc.arg(subject))
    RETURNING *
), created_message AS (
    INSERT INTO support_messages (thread_id, author_id, body)
    SELECT id, user_id, sqlc.arg(body)
    FROM created_thread
    RETURNING id, created_at
)
SELECT
    thread.id,
    thread.user_id,
    thread.subject,
    thread.status,
    thread.assigned_admin_id,
    thread.created_at,
    thread.updated_at,
    message.id AS first_message_id,
    message.created_at AS first_message_created_at
FROM created_thread AS thread
JOIN created_message AS message ON true;

-- name: ListUserSupportThreads :many
SELECT
    thread.id,
    thread.subject,
    thread.status,
    thread.assigned_admin_id,
    admin.nickname AS assigned_admin_nickname,
    thread.created_at,
    thread.updated_at,
    thread.closed_at,
    last_message.body AS last_message_body,
    last_message.created_at AS last_message_created_at,
    count(unread.id) AS unread_count
FROM support_threads AS thread
LEFT JOIN users AS admin
    ON admin.id = thread.assigned_admin_id
LEFT JOIN LATERAL (
    SELECT message.body, message.created_at
    FROM support_messages AS message
    WHERE message.thread_id = thread.id
    ORDER BY message.created_at DESC, message.id DESC
    LIMIT 1
) AS last_message ON true
LEFT JOIN support_messages AS unread
    ON unread.thread_id = thread.id
   AND unread.created_at > COALESCE(thread.user_read_at, '-infinity'::timestamptz)
   AND unread.author_id <> thread.user_id
WHERE thread.user_id = sqlc.arg(user_id)
GROUP BY thread.id, admin.nickname, last_message.body, last_message.created_at
ORDER BY thread.updated_at DESC, thread.id DESC;

-- name: GetUserSupportThread :one
SELECT
    thread.id,
    thread.user_id,
    thread.subject,
    thread.status,
    thread.assigned_admin_id,
    admin.nickname AS assigned_admin_nickname,
    thread.created_at,
    thread.updated_at,
    thread.closed_at
FROM support_threads AS thread
LEFT JOIN users AS admin ON admin.id = thread.assigned_admin_id
WHERE thread.id = sqlc.arg(thread_id)
  AND thread.user_id = sqlc.arg(user_id);

-- name: ListSupportMessages :many
SELECT
    message.id,
    message.thread_id,
    message.body,
    message.created_at,
    author.id AS author_id,
    author.nickname AS author_nickname,
    author.photo_url AS author_photo_url,
    author.is_admin AS author_is_admin
FROM support_messages AS message
JOIN users AS author ON author.id = message.author_id
WHERE message.thread_id = sqlc.arg(thread_id)
ORDER BY message.created_at, message.id;

-- name: CreateUserSupportMessage :one
WITH created AS (
    INSERT INTO support_messages (thread_id, author_id, body)
    SELECT thread.id, sqlc.arg(user_id), sqlc.arg(body)
    FROM support_threads AS thread
    WHERE thread.id = sqlc.arg(thread_id)
      AND thread.user_id = sqlc.arg(user_id)
      AND thread.status <> 'closed'
    RETURNING *
), touched AS (
    UPDATE support_threads AS thread
    SET updated_at = created.created_at,
        user_read_at = created.created_at
    FROM created
    WHERE thread.id = created.thread_id
)
SELECT
    created.id,
    created.thread_id,
    created.author_id,
    created.body,
    created.created_at
FROM created;

-- name: MarkSupportThreadReadByUser :execrows
UPDATE support_threads
SET user_read_at = clock_timestamp()
WHERE id = sqlc.arg(thread_id)
  AND user_id = sqlc.arg(user_id);

-- name: CloseSupportThreadByUser :execrows
UPDATE support_threads
SET status = 'closed',
    closed_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(thread_id)
  AND user_id = sqlc.arg(user_id)
  AND status <> 'closed';

-- name: ListAdminSupportThreads :many
SELECT
    thread.id,
    thread.user_id,
    customer.nickname AS user_nickname,
    customer.photo_url AS user_photo_url,
    thread.subject,
    thread.status,
    thread.assigned_admin_id,
    admin.nickname AS assigned_admin_nickname,
    thread.created_at,
    thread.updated_at,
    thread.closed_at,
    last_message.body AS last_message_body,
    last_message.created_at AS last_message_created_at,
    count(unread.id) AS unread_count
FROM support_threads AS thread
JOIN users AS customer ON customer.id = thread.user_id
LEFT JOIN users AS admin ON admin.id = thread.assigned_admin_id
LEFT JOIN LATERAL (
    SELECT message.body, message.created_at
    FROM support_messages AS message
    WHERE message.thread_id = thread.id
    ORDER BY message.created_at DESC, message.id DESC
    LIMIT 1
) AS last_message ON true
LEFT JOIN support_messages AS unread
    ON unread.thread_id = thread.id
   AND unread.created_at > COALESCE(thread.admin_read_at, '-infinity'::timestamptz)
   AND unread.author_id = thread.user_id
WHERE (sqlc.arg(status_filter)::text = '' OR thread.status::text = sqlc.arg(status_filter))
GROUP BY thread.id, customer.nickname, customer.photo_url, admin.nickname,
         last_message.body, last_message.created_at
ORDER BY
    CASE thread.status WHEN 'open' THEN 0 WHEN 'in_progress' THEN 1 ELSE 2 END,
    thread.updated_at DESC,
    thread.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountAdminSupportThreads :one
SELECT count(*)
FROM support_threads
WHERE (sqlc.arg(status_filter)::text = '' OR status::text = sqlc.arg(status_filter));

-- name: GetAdminSupportThread :one
SELECT
    thread.id,
    thread.user_id,
    customer.nickname AS user_nickname,
    customer.photo_url AS user_photo_url,
    thread.subject,
    thread.status,
    thread.assigned_admin_id,
    admin.nickname AS assigned_admin_nickname,
    thread.created_at,
    thread.updated_at,
    thread.closed_at
FROM support_threads AS thread
JOIN users AS customer ON customer.id = thread.user_id
LEFT JOIN users AS admin ON admin.id = thread.assigned_admin_id
WHERE thread.id = sqlc.arg(thread_id);

-- name: AssignSupportThread :execrows
UPDATE support_threads
SET assigned_admin_id = sqlc.arg(admin_id),
    status = 'in_progress',
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(thread_id)
  AND status = 'open'
  AND assigned_admin_id IS NULL;

-- name: CreateAdminSupportMessage :one
WITH created AS (
    INSERT INTO support_messages (thread_id, author_id, body)
    SELECT thread.id, sqlc.arg(admin_id), sqlc.arg(body)
    FROM support_threads AS thread
    WHERE thread.id = sqlc.arg(thread_id)
      AND thread.assigned_admin_id = sqlc.arg(admin_id)
      AND thread.status = 'in_progress'
    RETURNING *
), touched AS (
    UPDATE support_threads AS thread
    SET updated_at = created.created_at,
        admin_read_at = created.created_at
    FROM created
    WHERE thread.id = created.thread_id
)
SELECT
    created.id,
    created.thread_id,
    created.author_id,
    created.body,
    created.created_at
FROM created;

-- name: MarkSupportThreadReadByAdmin :execrows
UPDATE support_threads
SET admin_read_at = clock_timestamp()
WHERE id = sqlc.arg(thread_id);

-- name: CloseSupportThreadByAdmin :execrows
UPDATE support_threads
SET status = 'closed',
    closed_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(thread_id)
  AND assigned_admin_id = sqlc.arg(admin_id)
  AND status = 'in_progress';
