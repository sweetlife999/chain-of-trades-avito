import { memo, useState } from "react";
import axios from "axios";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import { deleteItem, getItem } from "../../../Api/items/items";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { Button } from "../../UI/Button/Button";
import { ConfirmationPopup } from "../../UI/ConfirmationPopup/ConfirmationPopup";
import { PhotoGallery } from "../../UI/PhotoGallery/PhotoGallery";
import { ItemPickupSection } from "../ItemPickupSection/ItemPickupSection";
import { ItemStatusBadge } from "../ItemStatusBadge/ItemStatusBadge";

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
            <PhotoGallery urls={item.photo_urls} alt={item.title} />
          </div>

          <div className={styles.item__info}>
            <ItemStatusBadge item={item} />
            <h1 className={styles.item__title}>{item.title}</h1>
            <small className={styles.item__category}>{item.category}</small>
            <p className={styles.item__description}>{item.description}</p>

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

      <ItemPickupSection isOwner={isOwner} item={item} />

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
