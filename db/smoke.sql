-- Проверка схемы на живой БД: собирает валидную цепочку обмена и убеждается, что БД
-- отвергает то, что должна отвергать. Всё внутри транзакции с ROLLBACK — база остаётся чистой.
-- Запуск: make smoke

\set ON_ERROR_STOP on

BEGIN;

-- Цикл из кейса: велосипед -> приставка -> смартфон -> велосипед
INSERT INTO users (id, nickname, password_hash) VALUES
    ('11111111-1111-1111-1111-111111111111', 'alice', 'hash'),
    ('22222222-2222-2222-2222-222222222222', 'bob',   'hash'),
    ('33333333-3333-3333-3333-333333333333', 'carol', 'hash');

INSERT INTO items (id, owner_id, category_id, title, photo_urls) VALUES
    ('aaaaaaaa-0000-0000-0000-000000000000',
     '11111111-1111-1111-1111-111111111111',
     (SELECT id FROM categories WHERE slug = 'bikes'),    'Велосипед',
     ARRAY['https://example.com/bike.jpg']),
    ('bbbbbbbb-0000-0000-0000-000000000000',
     '22222222-2222-2222-2222-222222222222',
     (SELECT id FROM categories WHERE slug = 'consoles'), 'Приставка',
     ARRAY['https://example.com/console-1.jpg', 'https://example.com/console-2.jpg']),
    ('cccccccc-0000-0000-0000-000000000000',
     '33333333-3333-3333-3333-333333333333',
     (SELECT id FROM categories WHERE slug = 'phones'),   'Смартфон',
     ARRAY['https://example.com/phone.jpg']);

-- за велосипед хотят приставку, за приставку — смартфон, за смартфон — велосипед
INSERT INTO item_wants (item_id, category_id) VALUES
    ('aaaaaaaa-0000-0000-0000-000000000000', (SELECT id FROM categories WHERE slug = 'consoles')),
    ('bbbbbbbb-0000-0000-0000-000000000000', (SELECT id FROM categories WHERE slug = 'phones')),
    ('cccccccc-0000-0000-0000-000000000000', (SELECT id FROM categories WHERE slug = 'bikes'));

INSERT INTO chains (id, signature, composition_key)
VALUES (
    'dddddddd-0000-0000-0000-000000000000',
    'smoke:dddddddd-0000-0000-0000-000000000000',
    'aaaaaaaa-0000-0000-0000-000000000000|bbbbbbbb-0000-0000-0000-000000000000|cccccccc-0000-0000-0000-000000000000'
);

INSERT INTO chain_participants (chain_id, user_id, gives_item_id, receives_item_id, position) VALUES
    ('dddddddd-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111',
     'aaaaaaaa-0000-0000-0000-000000000000', 'bbbbbbbb-0000-0000-0000-000000000000', 0),
    ('dddddddd-0000-0000-0000-000000000000', '22222222-2222-2222-2222-222222222222',
     'bbbbbbbb-0000-0000-0000-000000000000', 'cccccccc-0000-0000-0000-000000000000', 1),
    ('dddddddd-0000-0000-0000-000000000000', '33333333-3333-3333-3333-333333333333',
     'cccccccc-0000-0000-0000-000000000000', 'aaaaaaaa-0000-0000-0000-000000000000', 2);

-- 1. валидная цепочка собралась и она действительно замкнута
DO $$
DECLARE n int;
BEGIN
    SELECT count(*) INTO n FROM chain_participants
    WHERE chain_id = 'dddddddd-0000-0000-0000-000000000000';
    IF n <> 3 THEN
        RAISE EXCEPTION 'ожидали 3 участника, получили %', n;
    END IF;

    -- каждая отданная вещь кем-то получена: множества gives и receives совпадают
    SELECT count(*) INTO n FROM (
        SELECT gives_item_id AS i FROM chain_participants
        WHERE chain_id = 'dddddddd-0000-0000-0000-000000000000'
        EXCEPT
        SELECT receives_item_id FROM chain_participants
        WHERE chain_id = 'dddddddd-0000-0000-0000-000000000000'
    ) q;
    IF n <> 0 THEN
        RAISE EXCEPTION 'цепочка не замкнута: % вещей отдано, но не получено', n;
    END IF;
    RAISE NOTICE 'ok 1: валидная цепочка из 3 участников замкнута';
END $$;

-- 2. два участника на одной позиции в цепочке
DO $$
BEGIN
    INSERT INTO chain_participants (chain_id, user_id, gives_item_id, receives_item_id, position)
    VALUES ('dddddddd-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111',
            'cccccccc-0000-0000-0000-000000000000', 'bbbbbbbb-0000-0000-0000-000000000000', 0);
    RAISE EXCEPTION 'дубль position прошёл, а не должен был';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'ok 2: position уникален в пределах цепочки';
END $$;

-- 3. один и тот же пользователь дважды в одной цепочке
DO $$
BEGIN
    INSERT INTO chain_participants (chain_id, user_id, gives_item_id, receives_item_id, position)
    VALUES ('dddddddd-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111',
            'cccccccc-0000-0000-0000-000000000000', 'bbbbbbbb-0000-0000-0000-000000000000', 3);
    RAISE EXCEPTION 'дубль user_id прошёл, а не должен был';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'ok 3: пользователь входит в цепочку не более одного раза';
END $$;

-- 4. обмен вещи на саму себя
DO $$
BEGIN
    INSERT INTO chain_participants (chain_id, user_id, gives_item_id, receives_item_id, position)
    VALUES ('dddddddd-0000-0000-0000-000000000000', '22222222-2222-2222-2222-222222222222',
            'aaaaaaaa-0000-0000-0000-000000000000', 'aaaaaaaa-0000-0000-0000-000000000000', 4);
    RAISE EXCEPTION 'обмен вещи на саму себя прошёл, а не должен был';
EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'ok 4: gives_item_id <> receives_item_id';
END $$;

-- 5. рейтинг вне шкалы 0..5
DO $$
BEGIN
    UPDATE users SET rating = 6 WHERE nickname = 'alice';
    RAISE EXCEPTION 'рейтинг 6 прошёл, а не должен был';
EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'ok 5: рейтинг ограничен шкалой 0..5';
END $$;

-- 6. ник, отличающийся только регистром (логин ходит по nickname)
DO $$
BEGIN
    INSERT INTO users (nickname, password_hash) VALUES ('ALICE', 'hash');
    RAISE EXCEPTION 'ник ALICE прошёл при существующем alice';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'ok 6: nickname уникален регистронезависимо';
END $$;

-- 7. триггер set_updated_at перебивает то, что пишет запрос
DO $$
DECLARE ts timestamptz;
BEGIN
    UPDATE users SET updated_at = timestamptz '2000-01-01' WHERE nickname = 'alice';
    SELECT updated_at INTO ts FROM users WHERE nickname = 'alice';
    IF ts = timestamptz '2000-01-01' THEN
        RAISE EXCEPTION 'триггер set_updated_at не сработал, updated_at остался %', ts;
    END IF;
    RAISE NOTICE 'ok 7: триггер set_updated_at обновляет метку';
END $$;

-- 8. вещь нельзя привязать к несуществующей категории
DO $$
BEGIN
    INSERT INTO items (owner_id, category_id, title, photo_urls)
    VALUES ('11111111-1111-1111-1111-111111111111', 32767, 'Вещь из ниоткуда',
            ARRAY['https://example.com/nowhere.jpg']);
    RAISE EXCEPTION 'ссылка на несуществующую категорию прошла';
EXCEPTION WHEN foreign_key_violation THEN
    RAISE NOTICE 'ok 8: category_id проверяется внешним ключом';
END $$;

-- 9. объявление без фотографий
DO $$
BEGIN
    INSERT INTO items (owner_id, category_id, title, photo_urls)
    VALUES ('11111111-1111-1111-1111-111111111111',
            (SELECT id FROM categories WHERE slug = 'other'), 'Вещь без фото', ARRAY[]::text[]);
    RAISE EXCEPTION 'объявление с пустым списком фото прошло';
EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'ok 9: у объявления не меньше одной фотографии';
END $$;

-- 10. последнее фото нельзя удалить из существующего объявления
DO $$
BEGIN
    UPDATE items SET photo_urls = ARRAY[]::text[]
    WHERE id = 'aaaaaaaa-0000-0000-0000-000000000000';
    RAISE EXCEPTION 'удаление всех фотографий прошло';
EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'ok 10: последнюю фотографию удалить нельзя';
END $$;

-- 11. вещь, занятую в цепочке, удалить нельзя (иначе участник остался бы без предмета обмена)
DO $$
BEGIN
    DELETE FROM items WHERE id = 'aaaaaaaa-0000-0000-0000-000000000000';
    RAISE EXCEPTION 'удаление вещи из живой цепочки прошло';
EXCEPTION WHEN foreign_key_violation THEN
    RAISE NOTICE 'ok 11: вещь из цепочки защищена ON DELETE RESTRICT';
END $$;

-- 12. свободная вещь удаляется вместе со своими желаниями
DO $$
DECLARE n int;
BEGIN
    INSERT INTO items (id, owner_id, category_id, title, photo_urls)
    VALUES ('eeeeeeee-0000-0000-0000-000000000000',
            '11111111-1111-1111-1111-111111111111',
            (SELECT id FROM categories WHERE slug = 'books'), 'Книга',
            ARRAY['https://example.com/book.jpg']);
    INSERT INTO item_wants (item_id, category_id)
    VALUES ('eeeeeeee-0000-0000-0000-000000000000',
            (SELECT id FROM categories WHERE slug = 'tools'));

    DELETE FROM items WHERE id = 'eeeeeeee-0000-0000-0000-000000000000';

    SELECT count(*) INTO n FROM item_wants
    WHERE item_id = 'eeeeeeee-0000-0000-0000-000000000000';
    IF n <> 0 THEN
        RAISE EXCEPTION 'после удаления вещи осталось % желаний', n;
    END IF;
    RAISE NOTICE 'ok 12: item_wants уходят каскадом вместе с вещью';
END $$;

-- 13. в чужой тред писать нельзя (композитный ключ на chain_participants)
DO $$
DECLARE outsider uuid;
BEGIN
    INSERT INTO users (nickname, password_hash) VALUES ('dave', 'hash') RETURNING id INTO outsider;
    INSERT INTO chain_messages (chain_id, author_id, body)
    VALUES ('dddddddd-0000-0000-0000-000000000000', outsider, 'подвиньтесь, я тоже хочу');
    RAISE EXCEPTION 'сообщение от постороннего прошло, а не должно было';
EXCEPTION WHEN foreign_key_violation THEN
    RAISE NOTICE 'ok 13: писать в тред может только участник обмена';
END $$;

-- 14. событие обмена пишется без автора, тот же ключ ему не мешает
DO $$
DECLARE n int;
BEGIN
    INSERT INTO chain_messages (chain_id, kind)
    VALUES ('dddddddd-0000-0000-0000-000000000000', 'exchange_confirmed');

    SELECT count(*) INTO n FROM chain_messages
    WHERE chain_id = 'dddddddd-0000-0000-0000-000000000000' AND author_id IS NULL;
    IF n <> 1 THEN
        RAISE EXCEPTION 'ожидали одно событие без автора, получили %', n;
    END IF;
    RAISE NOTICE 'ok 14: событие обмена пишется без автора';
END $$;

-- 15. текстовое сообщение без текста
DO $$
BEGIN
    INSERT INTO chain_messages (chain_id, author_id, kind)
    VALUES ('dddddddd-0000-0000-0000-000000000000',
            '11111111-1111-1111-1111-111111111111', 'text');
    RAISE EXCEPTION 'текстовое сообщение без текста прошло';
EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'ok 15: у текстового сообщения обязателен текст';
END $$;

-- 16. обычная регистрация не должна случайно выдавать административные права
DO $$
DECLARE admin boolean;
BEGIN
    SELECT is_admin INTO admin FROM users WHERE nickname = 'alice';
    IF admin THEN
        RAISE EXCEPTION 'новый пользователь неожиданно получил права администратора';
    END IF;
    RAISE NOTICE 'ok 16: пользователь по умолчанию не администратор';
END $$;

-- 17. ПВЗ создаётся и обновляется, а updated_at выставляет общий триггер
DO $$
DECLARE
    point_id uuid;
    point_name text;
BEGIN
    INSERT INTO pickup_points (name, address)
    VALUES ('ПВЗ Центр', 'ул. Ленина, 10')
    RETURNING id INTO point_id;

    UPDATE pickup_points SET name = 'ПВЗ Север' WHERE id = point_id;
    SELECT name INTO point_name FROM pickup_points WHERE id = point_id;
    IF point_name <> 'ПВЗ Север' THEN
        RAISE EXCEPTION 'ПВЗ не обновился: %', point_name;
    END IF;
    RAISE NOTICE 'ok 17: ПВЗ создаётся и обновляется';
END $$;

-- 18. пустое название ПВЗ не проходит ограничение схемы
DO $$
BEGIN
    INSERT INTO pickup_points (name, address) VALUES ('   ', 'ул. Ленина, 10');
    RAISE EXCEPTION 'ПВЗ с пустым названием прошёл, а не должен был';
EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'ok 18: пустое название ПВЗ запрещено';
END $$;

-- 19. подпись занята, пока обмен открыт
DO $$
BEGIN
    INSERT INTO chains (signature, composition_key)
    VALUES ('smoke:dddddddd-0000-0000-0000-000000000000', 'smoke:different-composition');
    RAISE EXCEPTION 'второе открытое предложение с той же подписью прошло';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'ok 19: открытый обмен с той же подписью не создаётся';
END $$;

-- 19b. Другая перестановка того же набора вещей тоже не создаёт второй обмен.
DO $$
BEGIN
    INSERT INTO chains (signature, composition_key)
    VALUES (
        'smoke:reverse-direction',
        'aaaaaaaa-0000-0000-0000-000000000000|bbbbbbbb-0000-0000-0000-000000000000|cccccccc-0000-0000-0000-000000000000'
    );
    RAISE EXCEPTION 'второй активный обмен с тем же набором вещей прошёл';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'ok 19b: один набор вещей не создаёт разные активные перестановки';
END $$;

-- 20. отменённый обмен подпись не держит: иначе вытесненная цепочка не собралась бы
-- заново после срыва той, что её вытеснила. Кейс идёт последним: он меняет статус
-- цепочки, на которой стоят проверки выше.
DO $$
BEGIN
    UPDATE chains SET status = 'cancelled', closed_at = now()
    WHERE id = 'dddddddd-0000-0000-0000-000000000000';

    INSERT INTO chains (signature, composition_key, status, closed_at)
    VALUES (
        'smoke:dddddddd-0000-0000-0000-000000000000',
        'aaaaaaaa-0000-0000-0000-000000000000|bbbbbbbb-0000-0000-0000-000000000000|cccccccc-0000-0000-0000-000000000000',
        'cancelled',
        now()
    );
    INSERT INTO chains (signature, composition_key)
    VALUES (
        'smoke:dddddddd-0000-0000-0000-000000000000',
        'aaaaaaaa-0000-0000-0000-000000000000|bbbbbbbb-0000-0000-0000-000000000000|cccccccc-0000-0000-0000-000000000000'
    );

    RAISE NOTICE 'ok 20: отменённая подпись не мешает ни истории отмен, ни новому предложению';
END $$;

-- 21. повторная жалоба на то же сообщение от того же человека
-- Блок сам заводит сообщение: EXCEPTION откатывает всё, что блок успел сделать,
-- поэтому кейсы жалоб независимы и ничего друг другу не оставляют.
DO $$
DECLARE reported_message uuid;
BEGIN
    INSERT INTO chain_messages (chain_id, author_id, body)
    VALUES ('dddddddd-0000-0000-0000-000000000000',
            '22222222-2222-2222-2222-222222222222', 'встретимся у метро')
    RETURNING id INTO reported_message;

    INSERT INTO reports (reporter_id, message_id, reason)
    VALUES ('11111111-1111-1111-1111-111111111111', reported_message, 'spam');
    INSERT INTO reports (reporter_id, message_id, reason)
    VALUES ('11111111-1111-1111-1111-111111111111', reported_message, 'abuse');

    RAISE EXCEPTION 'вторая жалоба на то же сообщение прошла, а не должна была';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'ok 21: повторная жалоба на то же сообщение запрещена';
END $$;

-- 22. комментарий к жалобе длиннее 2000 символов
DO $$
DECLARE reported_message uuid;
BEGIN
    INSERT INTO chain_messages (chain_id, author_id, body)
    VALUES ('dddddddd-0000-0000-0000-000000000000',
            '22222222-2222-2222-2222-222222222222', 'встретимся у метро')
    RETURNING id INTO reported_message;

    INSERT INTO reports (reporter_id, message_id, reason, comment)
    VALUES ('11111111-1111-1111-1111-111111111111', reported_message, 'other',
            repeat('a', 2001));

    RAISE EXCEPTION 'комментарий длиннее 2000 символов прошёл, а не должен был';
EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'ok 22: комментарий к жалобе ограничен 2000 символами';
END $$;

-- 23. подпись занята и на этапе доставки. Обмен со своей подписью, чтобы не зависеть от
-- цепочки выше: она к этому моменту уже отменена кейсом 20.
DO $$
BEGIN
    INSERT INTO chains (signature, composition_key, status)
    VALUES ('smoke:delivering', 'smoke:delivering-composition', 'delivering');
    INSERT INTO chains (signature, composition_key)
    VALUES ('smoke:delivering', 'smoke:another-composition');
    RAISE EXCEPTION 'второе предложение с подписью доставляемого обмена прошло';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'ok 23: обмен в доставке держит свою подпись';
END $$;

-- 24. ПВЗ, в котором лежит вещь, удалить нельзя: иначе вещь потеряла бы адрес хранения
DO $$
DECLARE point_id uuid;
BEGIN
    INSERT INTO pickup_points (name, address) VALUES ('ПВЗ Смоук', 'ул. Тестовая, 1')
    RETURNING id INTO point_id;

    INSERT INTO items (owner_id, category_id, title, photo_urls, pickup_point_id)
    VALUES ('11111111-1111-1111-1111-111111111111',
            (SELECT id FROM categories WHERE slug = 'tools'), 'Дрель',
            ARRAY['https://example.com/drill.jpg'], point_id);

    DELETE FROM pickup_points WHERE id = point_id;
    RAISE EXCEPTION 'удаление ПВЗ с лежащей в нём вещью прошло';
EXCEPTION WHEN foreign_key_violation THEN
    RAISE NOTICE 'ok 24: ПВЗ с вещью защищён ON DELETE RESTRICT';
END $$;

-- 25. Причина закрытия предложения при снятии объявления поддерживается схемой треда.
DO $$
DECLARE n int;
BEGIN
    INSERT INTO chain_messages (chain_id, kind)
    VALUES ('dddddddd-0000-0000-0000-000000000000', 'exchange_item_withdrawn');

    SELECT count(*) INTO n
    FROM chain_messages
    WHERE chain_id = 'dddddddd-0000-0000-0000-000000000000'
      AND kind = 'exchange_item_withdrawn';
    IF n <> 1 THEN
        RAISE EXCEPTION 'ожидали одно событие снятия объявления, получили %', n;
    END IF;
    RAISE NOTICE 'ok 25: снятие объявления имеет отдельное событие треда';
END $$;

ROLLBACK;
