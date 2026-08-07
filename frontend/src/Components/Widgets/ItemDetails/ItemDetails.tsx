import { memo, useState } from "react";
import axios from "axios";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import { deleteItem, getItem } from "../../../Api/items/items";
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
    return <p>Загрузка...</p>;
  }

  if (isError || !item) {
    return <p>Не удалось загрузить вещь</p>;
  }

  const isOwner = item.owner_id === user?.id;

  return (
    <>
      <article className={styles.item}>
        <Link to="/myItems">← Мои вещи</Link>

        <div className={styles.item__grid}>
          <div className={styles.item__photos}>
            {item.photo_urls.map((url) => (
              <img key={url} src={url} alt={item.title} />
            ))}
          </div>

          <div className={styles.item__info}>
            <span className={styles[`status_${item.status}`]}>
              {labels[item.status]}
            </span>
            <h1>{item.title}</h1>
            <small>{item.category}</small>
            <p>{item.description}</p>

            <h2>Хочу получить</h2>
            <div className={styles.item__wants}>
              {item.wants.map((want) => (
                <span key={want}>{want}</span>
              ))}
            </div>

            {isOwner && (
              <div className={styles.item__actions}>
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
    </>
  );
};

export const ItemDetails = memo(ItemDetailsComponent);
