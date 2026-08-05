-- +goose Up

-- Объявление без фотографии бессмысленно, а «удалить все фото» — самый частый способ
-- получить пустую карточку. Требование «хотя бы одно фото» выражено CHECK-ом в БД:
-- ни один запрос мимо него не пройдёт, проверять в каждом хэндлере не нужно.
-- Массив вместо отдельной таблицы: порядок фото = порядок массива, и cardinality()
-- в CHECK видит весь список — из отдельной таблицы это выражалось бы только триггером.
ALTER TABLE items
    ADD COLUMN photo_urls text[];

-- Сначала переносим существующие ссылки. Старая схема разрешала NULL, поэтому записи
-- без фотографии сохраняем без выдуманных данных. NOT VALID не проверяет только старые
-- строки, но PostgreSQL применяет constraint ко всем новым INSERT/UPDATE.
UPDATE items
SET photo_urls = ARRAY[photo_url]
WHERE photo_url IS NOT NULL AND BTRIM(photo_url) <> '';

ALTER TABLE items
    ADD CONSTRAINT items_photo_urls_required
        CHECK (
            photo_urls IS NOT NULL
            AND cardinality(photo_urls) BETWEEN 1 AND 10
        ) NOT VALID,
    DROP COLUMN photo_url;

-- +goose Down

ALTER TABLE items
    ADD COLUMN photo_url text;

UPDATE items
SET photo_url = photo_urls[1]
WHERE cardinality(photo_urls) > 0;

ALTER TABLE items
    DROP COLUMN photo_urls;
