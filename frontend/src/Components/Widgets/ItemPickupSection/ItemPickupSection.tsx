import { memo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import {
  getItemErrorMessage,
  removeItemFromPickupPoint,
} from "../../../Api/items/items";
import type { TItem } from "../../../Api/items/items.types";
import { Button } from "../../UI/Button/Button";
import { ConfirmationPopup } from "../../UI/ConfirmationPopup/ConfirmationPopup";

type TProps = {
  isOwner: boolean;
  item: TItem;
};

const ItemPickupSectionComponent = ({ isOwner, item }: TProps) => {
  const [popupOpen, setPopupOpen] = useState(false);
  const queryClient = useQueryClient();

  const pickupMutation = useMutation({
    mutationFn: () => removeItemFromPickupPoint(item.id),
    onSuccess: async () => {
      setPopupOpen(false);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["items"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
      ]);
    },
  });

  const pickupPoint = item.pickup_point;
  if (!pickupPoint) {
    return null;
  }

  return (
    <>
      <section className={styles.pickup}>
        <span className={styles.pickup__label}>
          Вещь находится в пункте выдачи
        </span>
        <strong className={styles.pickup__name}>{pickupPoint.name}</strong>
        <span className={styles.pickup__address}>{pickupPoint.address}</span>

        {isOwner && (
          <Button
            className={styles.pickup__action}
            color="light"
            type="button"
            onClick={() => {
              pickupMutation.reset();
              setPopupOpen(true);
            }}
          >
            Забрать из ПВЗ
          </Button>
        )}
      </section>

      {popupOpen && (
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
              setPopupOpen(false);
            }
          }}
          onConfirm={() => pickupMutation.mutate()}
        />
      )}
    </>
  );
};

export const ItemPickupSection = memo(ItemPickupSectionComponent);
