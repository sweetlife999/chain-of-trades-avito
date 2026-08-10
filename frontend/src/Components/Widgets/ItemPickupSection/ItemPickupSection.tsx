import { memo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import clsx from "clsx";

import styles from "./Styles.module.scss";
import {
  getItemErrorMessage,
  removeItemFromPickupPoint,
} from "../../../Api/items/items";
import type { TItem } from "../../../Api/items/items.types";
import { Button } from "../../UI/Button/Button";
import { ConfirmationPopup } from "../../UI/ConfirmationPopup/ConfirmationPopup";
import { PickupPointSelector } from "../PickupPointSelector/PickupPointSelector";

type TProps = {
  isOwner: boolean;
  item: TItem;
};

type TRouteStep = {
  address?: string;
  name: string;
  state: "done" | "current" | "next";
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
    // Те же два статуса, что пропускает SetItemPickupPoint в queries/item_pickup.sql:
    // вещь вне оборота никуда не несут.
    const storable = item.status === "available" || item.status === "reserved";
    if (!isOwner || !storable) {
      return null;
    }

    return (
      <section className={styles.pickup}>
        <div className={styles.pickup__heading}>
          <h2 className={styles.pickup__title}>Где лежит вещь</h2>
          <p className={styles.pickup__description}>
            {item.status === "reserved"
              ? "Вещь у вас и ждёт передачи по обмену. Отнесите её в пункт выдачи, чтобы обмен поехал дальше."
              : "Вещь у вас. Отнести её в пункт выдачи можно заранее — к моменту сборки цепочки она уже будет на месте."}
          </p>
        </div>
        <PickupPointSelector currentPickupPoint={null} itemId={item.id} />
      </section>
    );
  }

  // Шагов два или три, и это само по себе информация: третий появляется, только когда у
  // вещи уже есть адресат.
  const route: TRouteStep[] = [
    { name: isOwner ? "У вас дома" : "У владельца", state: "done" },
    { address: pickupPoint.address, name: pickupPoint.name, state: "current" },
  ];
  if (item.status === "reserved") {
    route.push({ name: "У получателя", state: "next" });
  }

  return (
    <>
      <section className={styles.pickup}>
        <h2 className={styles.pickup__title}>Где лежит вещь</h2>

        <ol className={styles.pickup__route}>
          {route.map((step) => (
            <li
              aria-current={step.state === "current" ? "step" : undefined}
              className={clsx(
                styles.pickup__step,
                styles[`pickup__step_${step.state}`],
              )}
              key={step.name}
            >
              <span aria-hidden="true" className={styles.pickup__point} />
              <span className={styles.pickup__stepName}>{step.name}</span>
              {step.address && (
                <span className={styles.pickup__stepAddress}>
                  {step.address}
                </span>
              )}
            </li>
          ))}
        </ol>

        {isOwner && item.status === "available" && (
          <Button color="light" type="button" onClick={() => setPopupOpen(true)}>
            Забрать из ПВЗ
          </Button>
        )}

        {/* Забрать вещь из обмена нельзя: ClearPickupPoint вернёт 409 ErrItemInChain. */}
        {isOwner && item.status === "reserved" && (
          <p className={styles.pickup__hint}>
            Вещь участвует в обмене. Забрать её можно будет после завершения.
          </p>
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
