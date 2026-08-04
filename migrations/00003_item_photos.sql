-- +goose Up

-- Объявление без фотографии бессмысленно, а «удалить все фото» — самый частый способ
-- получить пустую карточку. Требование «хотя бы одно фото» выражено CHECK-ом в БД:
-- ни один запрос мимо него не пройдёт, проверять в каждом хэндлере не нужно.
-- Массив вместо отдельной таблицы: порядок фото = порядок массива, и cardinality()
-- в CHECK видит весь список — из отдельной таблицы это выражалось бы только триггером.
ALTER TABLE items
    DROP COLUMN photo_url,
    ADD COLUMN photo_urls text[] NOT NULL
        CHECK (cardinality(photo_urls) BETWEEN 1 AND 10);

-- +goose Down

ALTER TABLE items
    DROP COLUMN photo_urls,
    ADD COLUMN photo_url text;
