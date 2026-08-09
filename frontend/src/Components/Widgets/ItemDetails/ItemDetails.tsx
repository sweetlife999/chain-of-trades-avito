import { memo, useState } from "react";
import axios from "axios";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  deleteItem,
  getItem,
  getItemErrorMessage,
  removeItemFromPickupPoint,
} from "../../../Api/items/items";
import type { TItemStatus } from "../../../Api/items/items.types";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { Button } from "../../UI/Button/Button";
import { ConfirmationPopup } from "../../UI/ConfirmationPopup/ConfirmationPopup";

const labels: Record<TItemStatus, string> = {
  available: "Доступно для обмена",
  reserved: "Участвует в цепочке",
  traded: "Обменяно",
  withdrawn: "Снято с публикации",
};

const getDeleteErrorMessage = (error: unknown) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? "Не удалось удалить объявление";
  }

  return "Не удалось удалить объявление";
};

const ItemDetailsComponent = () => {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user } = useAuthSelector();
  const [deletePopupOpen, setDeletePopupOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string>();
  const [pickupPopupOpen, setPickupPopupOpen] = useState(false);

  const { data: item, isPending, isError } = useQuery({
    queryKey: ["items", id],
    queryFn: () => getItem(id),
    enabled: Boolean(id),
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteItem(id),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["items"] }),
        queryClient.invalidateQueries({ queryKey: ["posts"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
      ]);

      queryClient.removeQueries({ queryKey: ["items", id], exact: true });
      navigate("/myItems", { replace: true });
    },
    onError: (error) => {
      setDeleteError(getDeleteErrorMessage(error));
    },
  });

  const pickupMutation = useMutation({
    mutationFn: () => removeItemFromPickupPoint(id),
    onSuccess: async () => {
      setPickupPopupOpen(false);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["items"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
      ]);
    },
  });

  const openDeletePopup = () => {
    setDeleteError(undefined);
    setDeletePopupOpen(true);
  };

  const closeDeletePopup = () => {
    if (!deleteMutation.isPending) {
      setDeleteError(undefined);
      setDeletePopupOpen(false);
    }
  };

  if (isPending) {
    return <p className={styles.item__state}>Загружаем вещь...</p>;
  }

  if (isError || !item) {
    return (
      <p className={styles.item__state_error}>Не удалось загрузить вещь</p>
    );
  }

  const isOwner = item.owner_id === user?.id;

  return (
    <>
      <article className={styles.item}>
        <Link className={styles.item__back} to="/myItems">
          ← Мои вещи
        </Link>

        <div className={styles.item__grid}>
          <div className={styles.item__photos}>
            {item.photo_urls.map((url) => (
              <img
                className={styles.item__photo}
                key={url}
                src={url}
                alt={item.title}
              />
            ))}
          </div>

          <div className={styles.item__info}>
            <span
              className={`${styles.item__status} ${styles[`item__status_${item.status}`]}`}
            >
              {labels[item.status]}
            </span>
            <h1 className={styles.item__title}>{item.title}</h1>
            <small className={styles.item__category}>{item.category}</small>
            <p className={styles.item__description}>{item.description}</p>

            {item.pickup_point && (
              <div className={styles.item__pickup}>
                <span className={styles.item__pickupLabel}>
                  Вещь находится в пункте выдачи
                </span>
                <strong className={styles.item__pickupName}>
                  {item.pickup_point.name}
                </strong>
                <span className={styles.item__pickupAddress}>
                  {item.pickup_point.address}
                </span>
              </div>
            )}

            <h2 className={styles.item__wantsTitle}>Хочу получить</h2>
            <div className={styles.item__wants}>
              {item.wants.map((want) => (
                <span className={styles.item__want} key={want}>
                  {want}
                </span>
              ))}
            </div>

            {isOwner && (
              <div className={styles.item__actions}>
                {item.pickup_point && (
                  <Button
                    color="light"
                    type="button"
                    onClick={() => {
                      pickupMutation.reset();
                      setPickupPopupOpen(true);
                    }}
                  >
                    Забрать из ПВЗ
                  </Button>
                )}
                <Button
                  color="light"
                  type="button"
                  onClick={() => navigate(`/items/${item.id}/edit`)}
                >
                  Редактировать
                </Button>

                <Button color="danger" type="button" onClick={openDeletePopup}>
                  Удалить
                </Button>
              </div>
            )}
          </div>
        </div>
      </article>

      {deletePopupOpen && (
        <ConfirmationPopup
          title="Удалить объявление?"
          description={`Объявление «${item.title}» будет удалено. Это действие нельзя отменить.`}
          confirmLabel="Удалить"
          pendingLabel="Удаляем..."
          isPending={deleteMutation.isPending}
          error={deleteError}
          onClose={closeDeletePopup}
          onConfirm={() => deleteMutation.mutate()}
        />
      )}

      {pickupPopupOpen && item.pickup_point && (
        <ConfirmationPopup
          title="Забрать вещь из ПВЗ?"
          description={`«${item.title}» вернётся домой. Если вещь участвует в незавершённом обмене, сервер не позволит её забрать.`}
          confirmLabel="Забрать вещь"
          pendingLabel="Возвращаем..."
          isPending={pickupMutation.isPending}
          error={
            pickupMutation.isError
              ? getItemErrorMessage(
                  pickupMutation.error,
                  "Не удалось забрать вещь из ПВЗ.",
                )
              : undefined
          }
          onClose={() => {
            if (!pickupMutation.isPending) {
              pickupMutation.reset();
              setPickupPopupOpen(false);
            }
          }}
          onConfirm={() => pickupMutation.mutate()}
        />
      )}
    </>
  );
};

export const ItemDetails = memo(ItemDetailsComponent);
