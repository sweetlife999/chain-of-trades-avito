-- Блокировка и запись аудита выполняются одним SQL statement: либо сохранятся обе,
-- либо не сохранится ничего. Самого себя администратор заблокировать не может.
-- name: BlockUserForAdmin :one
WITH changed AS (
    UPDATE users AS account
    SET
        is_blocked = true,
        blocked_at = now(),
        blocked_by = sqlc.arg(admin_id)
    WHERE account.id = sqlc.arg(user_id)
      AND account.id <> sqlc.arg(admin_id)
      AND NOT account.is_blocked
    RETURNING account.id, account.nickname, account.is_blocked, account.blocked_at, account.blocked_by
), logged AS (
    INSERT INTO admin_audit_log (admin_id, action, target_type, target_id)
    SELECT sqlc.arg(admin_id), 'user_blocked', 'user', changed.id
    FROM changed
    RETURNING target_id
)
SELECT changed.id, changed.nickname, changed.is_blocked, changed.blocked_at, changed.blocked_by
FROM changed
JOIN logged ON logged.target_id = changed.id;

-- name: UnblockUserForAdmin :one
WITH changed AS (
    UPDATE users AS account
    SET
        is_blocked = false,
        blocked_at = NULL,
        blocked_by = NULL
    WHERE account.id = sqlc.arg(user_id)
      AND account.is_blocked
    RETURNING account.id, account.nickname, account.is_blocked, account.blocked_at, account.blocked_by
), logged AS (
    INSERT INTO admin_audit_log (admin_id, action, target_type, target_id)
    SELECT sqlc.arg(admin_id), 'user_unblocked', 'user', changed.id
    FROM changed
    RETURNING target_id
)
SELECT changed.id, changed.nickname, changed.is_blocked, changed.blocked_at, changed.blocked_by
FROM changed
JOIN logged ON logged.target_id = changed.id;

-- Используется middleware на каждом защищённом запросе. Поэтому глобальная
-- блокировка немедленно отзывает и уже выданный JWT.
-- name: CanUserAuthenticate :one
SELECT EXISTS (
    SELECT 1
    FROM users AS account
    WHERE account.id = sqlc.arg(user_id)
      AND NOT account.is_blocked
);

-- name: CreateAdminAuditLog :one
INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, metadata)
VALUES (
    sqlc.arg(admin_id),
    sqlc.arg(action),
    sqlc.arg(target_type),
    sqlc.arg(target_id),
    sqlc.arg(metadata)
)
RETURNING *;

-- name: ListAdminAuditLog :many
SELECT log.*
FROM admin_audit_log AS log
WHERE (sqlc.narg(admin_id)::uuid IS NULL OR log.admin_id = sqlc.narg(admin_id)::uuid)
  AND (sqlc.arg(action)::text = '' OR log.action::text = sqlc.arg(action)::text)
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR log.created_at >= sqlc.narg(date_from)::timestamptz)
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR log.created_at <= sqlc.narg(date_to)::timestamptz)
ORDER BY log.created_at DESC, log.id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountAdminAuditLog :one
SELECT count(*)
FROM admin_audit_log AS log
WHERE (sqlc.narg(admin_id)::uuid IS NULL OR log.admin_id = sqlc.narg(admin_id)::uuid)
  AND (sqlc.arg(action)::text = '' OR log.action::text = sqlc.arg(action)::text)
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR log.created_at >= sqlc.narg(date_from)::timestamptz)
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR log.created_at <= sqlc.narg(date_to)::timestamptz);
