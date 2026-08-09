-- Владелец в WHERE, а не проверкой в Go: чужую вещь запрос просто не найдёт. :execrows
-- различает «обновили» и «нечего обновлять», и вызывающий раскладывает ноль в 403/409.
-- Статусы traded и withdrawn отсекаются здесь же: вещь вне оборота никуда не несут, а
-- отметку завершённого обмена снимает ClearExchangeItemsPickupPoint.
-- name: SetItemPickupPoint :execrows
UPDATE items
SET pickup_point_id = sqlc.arg(pickup_point_id)
WHERE id = sqlc.arg(item_id)
  AND owner_id = sqlc.arg(owner_id)
  AND status IN ('available', 'reserved');

-- name: ClearItemPickupPoint :execrows
UPDATE items
SET pickup_point_id = NULL
WHERE id = sqlc.arg(item_id)
  AND owner_id = sqlc.arg(owner_id)
  AND status IN ('available', 'reserved');
