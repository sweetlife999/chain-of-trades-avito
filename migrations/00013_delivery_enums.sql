-- +goose Up

-- Значения enum приезжают отдельной миграцией и здесь же не используются: Postgres не
-- даёт применить новое значение в той же транзакции, где оно добавлено, а goose катает
-- миграцию транзакцией. Всё, что на эти значения ссылается, лежит в 00014.
--
-- IF NOT EXISTS обязателен, потому что Down ниже — no-op: после отката до 00012 и
-- повторного наката без него ALTER упал бы на уже существующем значении.

-- Промежуток между «все согласились» и «все получили» разрезается надвое: вещи сданы в
-- пункты (delivering) и пункты передали их получателям (delivered). Второй переход
-- делает администратор, поэтому в бэкенде на него пока никто не переводит.
ALTER TYPE chain_status ADD VALUE IF NOT EXISTS 'delivering' AFTER 'confirmed';
ALTER TYPE chain_status ADD VALUE IF NOT EXISTS 'delivered' AFTER 'delivering';

-- Тред — единственный канал, которым участник узнаёт о смене состояния, поэтому у
-- каждого нового перехода есть событие. participant_delivered_item отвечает на вопрос
-- «кого ещё ждём»: у него есть автор, у событий всей цепи автора нет.
ALTER TYPE chain_message_kind ADD VALUE IF NOT EXISTS 'participant_delivered_item';
ALTER TYPE chain_message_kind ADD VALUE IF NOT EXISTS 'exchange_delivering';
ALTER TYPE chain_message_kind ADD VALUE IF NOT EXISTS 'exchange_delivered';

-- +goose Down

-- ponytail: откат пустой. DROP VALUE в Postgres нет, а честный откат — пересоздать оба
-- типа и переписать все зависимые колонки, включая уже накопленные строки chain_messages.
-- Лишнее значение в enum ничему не мешает: на него никто не переводит после отката 00014.
SELECT 1;
