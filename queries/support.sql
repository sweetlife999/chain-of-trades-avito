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
    -- COALESCE, потому что sqlc типизирует эту колонку как NOT NULL по исходному
    -- столбцу и не учитывает LEFT JOIN: NULL здесь — не пустое превью, а 500 на
    -- весь список.
    COALESCE(last_message.body, '') AS last_message_body,
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
        user_read_at = created.created_at,
        -- Сюда попадает только повторное сообщение: первое вставляет CreateSupportThread.
        -- Значит автоответ не решил вопрос и обращению нужен человек. COALESCE держит
        -- метку на моменте первой эскалации, а не последнего сообщения.
        escalated_at = COALESCE(thread.escalated_at, created.created_at)
    FROM created
    WHERE thread.id = created.thread_id
)
SELECT
    created.id,
    created.thread_id,
    created.author_id,
    created.body,
    created.created_at,
    author.nickname AS author_nickname,
    author.photo_url AS author_photo_url,
    author.is_admin AS author_is_admin
FROM created
JOIN users AS author ON author.id = created.author_id;

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
    thread.escalated_at,
    -- COALESCE, потому что sqlc типизирует эту колонку как NOT NULL по исходному
    -- столбцу и не учитывает LEFT JOIN: NULL здесь — не пустое превью, а 500 на
    -- весь список.
    COALESCE(last_message.body, '') AS last_message_body,
    last_message.created_at AS last_message_created_at,
    count(unread.id) AS unread_count
FROM support_threads AS thread
JOIN users AS customer ON customer.id = thread.user_id
LEFT JOIN users AS admin ON admin.id = thread.assigned_admin_id
LEFT JOIN LATERAL (
    SELECT message.body, message.created_at
    FROM support_messages AS message
    WHERE message.thread_id = thread.id
      -- В очереди модерации превью — последнее сообщение автора обращения, а не ответ
      -- поддержки: иначе автоответчик, который отвечает почти сразу, сделал бы превью
      -- всех обращений одинаковым и спрятал бы из очереди сам вопрос.
      --
      -- Условие именно «автор обращения», а не «не администратор»: администратор тоже
      -- обычный пользователь и может написать в поддержку сам. По «не администратор»
      -- такой тред не давал ни одной строки, а NULL в last_message_body ронял 500 на
      -- всю страницу очереди, потому что sqlc типизирует эту колонку как NOT NULL.
      -- Тот же признак стоит у подсчёта непрочитанного ниже.
      AND message.author_id = thread.user_id
    ORDER BY message.created_at DESC, message.id DESC
    LIMIT 1
) AS last_message ON true
LEFT JOIN support_messages AS unread
    ON unread.thread_id = thread.id
   AND unread.created_at > COALESCE(thread.admin_read_at, '-infinity'::timestamptz)
   AND unread.author_id = thread.user_id
WHERE (sqlc.arg(status_filter)::text = '' OR thread.status::text = sqlc.arg(status_filter))
  -- needs_human оставляет только то, что действительно ждёт модератора: обращения с
  -- отметкой эскалации плюс те, на которые со стороны сервиса вообще никто не ответил.
  -- Вторая половина обязательна: без неё из очереди пропали бы обращения, пришедшие
  -- когда автоответчик был выключен, недоступен или ещё не существовал, — то есть
  -- поломка модели прятала бы обращения вместо того, чтобы их показывать.
  AND (
      NOT sqlc.arg(needs_human)::boolean
      OR thread.escalated_at IS NOT NULL
      OR NOT EXISTS (
          SELECT 1
          FROM support_messages AS answer
          JOIN users AS responder ON responder.id = answer.author_id
          WHERE answer.thread_id = thread.id
            AND responder.is_admin
      )
  )
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
FROM support_threads AS thread
WHERE (sqlc.arg(status_filter)::text = '' OR thread.status::text = sqlc.arg(status_filter))
  -- needs_human оставляет только то, что действительно ждёт модератора: обращения с
  -- отметкой эскалации плюс те, на которые со стороны сервиса вообще никто не ответил.
  -- Вторая половина обязательна: без неё из очереди пропали бы обращения, пришедшие
  -- когда автоответчик был выключен, недоступен или ещё не существовал, — то есть
  -- поломка модели прятала бы обращения вместо того, чтобы их показывать.
  AND (
      NOT sqlc.arg(needs_human)::boolean
      OR thread.escalated_at IS NOT NULL
      OR NOT EXISTS (
          SELECT 1
          FROM support_messages AS answer
          JOIN users AS responder ON responder.id = answer.author_id
          WHERE answer.thread_id = thread.id
            AND responder.is_admin
      )
  );

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
    thread.closed_at,
    thread.escalated_at
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
    created.created_at,
    author.nickname AS author_nickname,
    author.photo_url AS author_photo_url,
    author.is_admin AS author_is_admin
FROM created
JOIN users AS author ON author.id = created.author_id;

-- Ответ автоответчика. Отдельный запрос, а не CreateAdminSupportMessage, по трём
-- причинам: тот требует назначенного администратора и статус in_progress, а бот не
-- берёт обращение на себя (иначе тред ушёл бы из очереди «открытые»); тот ставит
-- admin_read_at, то есть погасил бы непрочитанное у живых модераторов; и updated_at
-- бот намеренно не двигает, чтобы порядок очереди не зависел от его ответа.
--
-- Автор находится джойном по нику из миграции 00024, поэтому UUID служебного
-- пользователя в Go не хранится. Ноль строк — нормальный ответ: значит обращение
-- закрыли, взяли в работу или бот в нём уже отвечал.
-- name: CreateBotSupportMessage :one
WITH created AS (
    INSERT INTO support_messages (thread_id, author_id, body)
    SELECT thread.id, bot.id, sqlc.arg(body)
    FROM support_threads AS thread
    JOIN users AS bot ON lower(bot.nickname) = lower(sqlc.arg(bot_nickname)::text)
    WHERE thread.id = sqlc.arg(thread_id)
      -- Только пока обращение никто не взял: администратор мог назначить его на себя,
      -- пока считала модель.
      AND thread.status = 'open'
      -- Один автоответ на обращение. Повтор отсекается здесь, а не проверкой перед
      -- вставкой, — тот же приём, что UNIQUE у жалоб.
      AND NOT EXISTS (
          SELECT 1
          FROM support_messages AS previous
          WHERE previous.thread_id = thread.id
            AND previous.author_id = bot.id
      )
    RETURNING *
)
SELECT
    created.id,
    created.thread_id,
    created.author_id,
    created.body,
    created.created_at,
    author.nickname AS author_nickname,
    author.photo_url AS author_photo_url,
    author.is_admin AS author_is_admin
FROM created
JOIN users AS author ON author.id = created.author_id;

-- Совпадает ли ник служебного пользователя в коде с ником из миграции. Проверяется на
-- старте: разъехавшись, они дают молчание бота на каждом обращении вместо ошибки.
-- name: SupportBotUserExists :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE lower(nickname) = lower(sqlc.arg(bot_nickname)::text)
      AND is_admin
);

-- Пометить, что обращению нужен человек. Вызывает автоответчик, когда не справился:
-- модель отнесла обращение к теме `other` или не ответила вовсе. Идемпотентно — повторный
-- вызов метку не двигает, а закрытое обращение не эскалирует.
-- name: EscalateSupportThread :execrows
UPDATE support_threads
SET escalated_at = clock_timestamp()
WHERE id = sqlc.arg(thread_id)
  AND escalated_at IS NULL
  AND status <> 'closed';

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
