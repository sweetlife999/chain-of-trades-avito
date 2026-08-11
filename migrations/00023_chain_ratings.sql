-- +goose Up

-- Средний балл лежит в users.rating с самой первой миграции и до сих пор никем не писался.
-- Рядом появляется счётчик: без него «оценок ещё нет» неотличимо от честного нуля, а
-- ровно это различие users.rating и защищает своим NULL.
ALTER TABLE users
    ADD COLUMN ratings_count integer NOT NULL DEFAULT 0 CHECK (ratings_count >= 0);

-- Оценивают того, от кого пришла вещь: в цикле это участник, чья отдаваемая вещь совпала
-- с моей получаемой. Поэтому оценка привязана к цепочке, а не к паре пользователей — одну
-- и ту же пару может свести несколько разных обменов, и каждый заслуживает своей оценки.
--
-- Обе стороны ссылаются на chain_participants (chain_id, user_id), а не на users: там уже
-- есть нужный UNIQUE, и «оценить можно только внутри цепочки, в которой оба участвуют»
-- становится фактом базы, а не проверкой в сервисе. Транзитивно это и ссылка на users,
-- поэтому отдельный FK туда был бы вторым забором вокруг того же поля.
--
-- ON DELETE CASCADE достижим только через DELETE FROM chains: участников не удаляют
-- (chain_participants.user_id — RESTRICT), отказ меняет статус. Так чистятся за собой
-- интеграционные тесты, и ровно поэтому триггер ниже обрабатывает DELETE.
CREATE TABLE chain_ratings (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id   uuid NOT NULL,
    rater_id   uuid NOT NULL,
    rated_id   uuid NOT NULL,
    score      smallint NOT NULL CHECK (score BETWEEN 1 AND 5),
    -- Столько же, сколько у сообщения обмена и комментария к жалобе. Пустой комментарий —
    -- это '', а не NULL: «не написал ничего» и «написал пустоту» тут одно и то же.
    comment    text NOT NULL DEFAULT '' CHECK (char_length(comment) <= 2000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- Одна оценка на участника в цепочке. Он же арбитр ON CONFLICT для перезаписи оценки
    -- и лукап «моя оценка в этом обмене», поэтому отдельного индекса под них не нужно.
    UNIQUE (chain_id, rater_id),
    -- Партнёр выводится джойном по вещам, а не приходит от клиента. CHECK ловит тот
    -- единственный случай, когда этот вывод сломается.
    CHECK  (rater_id <> rated_id),
    FOREIGN KEY (chain_id, rater_id) REFERENCES chain_participants (chain_id, user_id) ON DELETE CASCADE,
    FOREIGN KEY (chain_id, rated_id) REFERENCES chain_participants (chain_id, user_id) ON DELETE CASCADE
);

-- Лента отзывов о человеке: от новых к старым, страницами. id в сортировке — случайный
-- uuid, он лишь разводит одинаковые created_at и порядок вставки не выдаёт.
CREATE INDEX chain_ratings_rated_created_idx
    ON chain_ratings (rated_id, created_at DESC, id DESC);

CREATE TRIGGER chain_ratings_set_updated_at
BEFORE UPDATE ON chain_ratings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Средний балл и счётчик держит триггер, как и уведомления: производная от таблицы живёт
-- рядом с таблицей и не зависит от того, каким путём пришла запись. Иначе запись оценки,
-- которая иначе один statement, потребовала бы транзакции и второго модуля, пишущего users.
--
-- Пересчёт целиком, а не дельтой: rating уже округлён до numeric(3,2), поэтому
-- rating * count ± score дрейфует, а при правке оценки ему нужен ещё и прежний балл.
-- Пересчёт по одному индексу идемпотентен и стоит столько же.
-- +goose StatementBegin
CREATE FUNCTION refresh_user_rating() RETURNS trigger AS $$
DECLARE
    target uuid := CASE WHEN TG_OP = 'DELETE' THEN OLD.rated_id ELSE NEW.rated_id END;
BEGIN
    -- Блокировка профиля отдельным statement'ом, до чтения оценок. Под READ COMMITTED
    -- подзапрос внутри UPDATE ... FROM (SELECT ...) считается на снапшоте начала запроса,
    -- и конкурент, вставивший оценку тому же человеку, потерялся бы в счётчике: ожидание
    -- на блокировке строки агрегат заново не выполняет. Следующий statement берёт снапшот
    -- уже после его коммита. Склеивать эти две команды в одну нельзя.
    PERFORM 1 FROM users WHERE id = target FOR UPDATE;

    UPDATE users
    SET rating        = aggregate.average,
        ratings_count = aggregate.total
    FROM (
        SELECT round(avg(score), 2) AS average, count(*)::integer AS total
        FROM chain_ratings
        WHERE rated_id = target
    ) AS aggregate
    WHERE users.id = target;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- UPDATE OF score, а не UPDATE: правка комментария рейтинг не двигает. Лишние срабатывания
-- (upsert всегда пишет score, даже тот же самый) безвредны — пересчёт идемпотентен.
CREATE TRIGGER chain_ratings_refresh_user_rating
AFTER INSERT OR UPDATE OF score OR DELETE ON chain_ratings
FOR EACH ROW EXECUTE FUNCTION refresh_user_rating();

-- +goose Down

DROP TRIGGER chain_ratings_refresh_user_rating ON chain_ratings;
DROP FUNCTION refresh_user_rating();

DROP TRIGGER chain_ratings_set_updated_at ON chain_ratings;

DROP TABLE chain_ratings;

ALTER TABLE users DROP COLUMN ratings_count;

-- До этой миграции rating был NULL у всех: писать его было нечем. Оставить посчитанные
-- значения без таблицы, из которой они выведены, значит оставить число, которое уже
-- никто не пересчитает.
UPDATE users SET rating = NULL WHERE rating IS NOT NULL;
