-- Кого оценивают, клиент не передаёт: партнёр выводится из самого цикла — это участник,
-- чья отдаваемая вещь пришла ко мне. Меньше полей в запросе, и целый класс проверок
-- («а можно ли оценивать этого?») перестаёт существовать.
--
-- Все условия висят на SELECT, который питает INSERT. Когда он отдаёт ноль строк, INSERT
-- не выполняется, а значит и ON CONFLICT не срабатывает: просроченное окно и незавершённый
-- обмен блокируют не только первую оценку, но и правку уже поставленной.
--
-- Ноль строк на выходе означает «одна из проверок не прошла» и не говорит какая:
-- разбирается в этом отдельный GetExchangeRatingEligibility, и только когда что-то пошло
-- не так. Счастливый путь остаётся одним запросом.
-- name: UpsertExchangeRating :one
INSERT INTO chain_ratings (chain_id, rater_id, rated_id, score, comment)
SELECT
    rater.chain_id,
    rater.user_id,
    rated.user_id,
    sqlc.arg(score)::smallint,
    sqlc.arg(comment)::text
FROM chain_participants AS rater
JOIN chains AS exchange
    ON exchange.id = rater.chain_id
JOIN chain_participants AS rated
    ON rated.chain_id = rater.chain_id
   AND rated.gives_item_id = rater.receives_item_id
WHERE rater.chain_id = sqlc.arg(exchange_id)
  AND rater.user_id = sqlc.arg(rater_id)
  AND exchange.status = 'completed'
  -- closed_at IS NULL даёт false и закрывает окно: у завершённого обмена он всегда есть,
  -- а если когда-то не окажется — отказ безопаснее вечно открытой оценки.
  AND exchange.closed_at > now() - interval '14 days'
ON CONFLICT (chain_id, rater_id) DO UPDATE
SET score   = EXCLUDED.score,
    comment = EXCLUDED.comment
RETURNING rated_id, score, comment, created_at, updated_at;

-- Зачем оценка не прошла: обмена нет (ноль строк), не участник, не завершён или окно
-- закрылось. Нужен только для выбора кода ответа после неудачного upsert'а, поэтому
-- ничего сверх этого не читает — состояние оценки клиент и так видит в самом обмене.
--
-- Окно приезжает готовым ответом, а не сроком: сравнивал бы его Go — в проверке
-- появились бы вторые часы, расходящиеся с теми, по которым отказал сам upsert.
-- name: GetExchangeRatingEligibility :one
SELECT
    exchange.status AS exchange_status,
    (exchange.closed_at > now() - interval '14 days')::boolean AS window_open,
    (rater.user_id IS NOT NULL)::boolean AS is_participant
FROM chains AS exchange
LEFT JOIN chain_participants AS rater
    ON rater.chain_id = exchange.id
   AND rater.user_id = sqlc.arg(rater_id)
WHERE exchange.id = sqlc.arg(exchange_id);

-- Лента отзывов о человеке. Анонимная: ни автора, ни цепочки, ни id самой оценки, ни
-- времени правки — всё это каналы, по которым автор вычисляется. Общее число не считаем:
-- оно уже лежит в users.ratings_count того же профиля.
-- name: ListUserRatings :many
SELECT score, comment, created_at
FROM chain_ratings
WHERE rated_id = sqlc.arg(rated_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);
