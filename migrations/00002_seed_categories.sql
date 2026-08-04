-- +goose Up

INSERT INTO categories (slug, name) VALUES
    ('electronics', 'Электроника'),
    ('phones',      'Смартфоны'),
    ('consoles',    'Игровые приставки'),
    ('computers',   'Компьютеры и комплектующие'),
    ('bikes',       'Велосипеды и транспорт'),
    ('sports',      'Спорт и отдых'),
    ('books',       'Книги'),
    ('clothes',     'Одежда и обувь'),
    ('furniture',   'Мебель и интерьер'),
    ('tools',       'Инструменты'),
    ('hobby',       'Хобби и творчество'),
    ('other',       'Прочее')
ON CONFLICT (slug) DO NOTHING;

-- +goose Down

DELETE FROM categories WHERE slug IN (
    'electronics', 'phones', 'consoles', 'computers', 'bikes', 'sports',
    'books', 'clothes', 'furniture', 'tools', 'hobby', 'other'
);
