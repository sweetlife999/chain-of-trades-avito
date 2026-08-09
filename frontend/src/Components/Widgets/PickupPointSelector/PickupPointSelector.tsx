import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, MapPin } from "lucide-react";

import {
  getItemErrorMessage,
  setItemPickupPoint,
} from "../../../Api/items/items";
import {
  getPickupPoint,
  getPickupPoints,
} from "../../../Api/pickupPoints/pickupPoints";
import type { TItemPickupPoint } from "../../../Api/items/items.types";
import { Button } from "../../UI/Button/Button";
import { Loader } from "../../UI/Loader/Loader";
import styles from "./Styles.module.scss";

type TProps = {
  currentPickupPoint?: TItemPickupPoint | null;
  itemId: string;
};

export const PickupPointSelector = ({
  currentPickupPoint,
  itemId,
}: TProps) => {
  const [selectedId, setSelectedId] = useState(currentPickupPoint?.id ?? "");
  const queryClient = useQueryClient();

  const pickupPointsQuery = useQuery({
    queryKey: ["pickup-points"],
    queryFn: getPickupPoints,
    enabled: !currentPickupPoint,
    staleTime: 60_000,
  });

  const pickupPointQuery = useQuery({
    queryKey: ["pickup-points", selectedId],
    queryFn: () => getPickupPoint(selectedId),
    enabled: Boolean(selectedId),
    retry: false,
  });

  const pickupMutation = useMutation({
    mutationFn: (pickupPointId: string) =>
      setItemPickupPoint(itemId, { pickup_point_id: pickupPointId }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
        queryClient.invalidateQueries({ queryKey: ["items"] }),
      ]);
    },
  });

  if (currentPickupPoint) {
    const pickupPoint = pickupPointQuery.data ?? currentPickupPoint;

    return (
      <div className={styles.selector__delivered} role="status">
        <CheckCircle2
          aria-hidden="true"
          className={styles.selector__deliveredIcon}
        />
        <div className={styles.selector__deliveredContent}>
          <strong className={styles.selector__deliveredTitle}>
            Ваша вещь уже находится в ПВЗ
          </strong>
          <span className={styles.selector__deliveredName}>
            {pickupPoint.name}
          </span>
          <span className={styles.selector__deliveredAddress}>
            {pickupPoint.address}
          </span>
          <p className={styles.selector__lockHint}>
            Пока обмен не завершён, забрать вещь нельзя — на неё рассчитывают
            остальные участники.
          </p>
        </div>
      </div>
    );
  }

  const pickupPoints = pickupPointsQuery.data ?? [];

  return (
    <div className={styles.selector}>
      <div className={styles.selector__heading}>
        <span className={styles.selector__icon} aria-hidden="true">
          <MapPin className={styles.selector__iconImage} />
        </span>
        <div className={styles.selector__headingText}>
          <h3 className={styles.selector__title}>Выберите пункт выдачи</h3>
          <p className={styles.selector__description}>
            Нажмите на ПВЗ, проверьте его адрес и подтвердите передачу вещи.
          </p>
        </div>
      </div>

      {pickupPointsQuery.isPending && (
        <div className={styles.selector__state}>
          <Loader text="Загружаем пункты выдачи..." />
        </div>
      )}

      {pickupPointsQuery.isError && (
        <div className={styles.selector__state} role="alert">
          <strong className={styles.selector__stateTitle}>
            Не удалось загрузить ПВЗ
          </strong>
          <Button
            color="light"
            size="s"
            onClick={() => void pickupPointsQuery.refetch()}
          >
            Повторить
          </Button>
        </div>
      )}

      {!pickupPointsQuery.isPending &&
        !pickupPointsQuery.isError &&
        pickupPoints.length === 0 && (
          <p className={styles.selector__state}>
            Доступных пунктов выдачи пока нет. Сообщите администратору.
          </p>
        )}

      {pickupPoints.length > 0 && (
        <div className={styles.selector__list}>
          {pickupPoints.map((pickupPoint) => (
            <button
              aria-pressed={selectedId === pickupPoint.id}
              className={[
                styles.selector__option,
                selectedId === pickupPoint.id
                  ? styles.selector__option_selected
                  : "",
              ]
                .filter(Boolean)
                .join(" ")}
              key={pickupPoint.id}
              onClick={() => {
                pickupMutation.reset();
                setSelectedId(pickupPoint.id);
              }}
              type="button"
            >
              <span className={styles.selector__optionMarker} aria-hidden="true" />
              <span className={styles.selector__optionContent}>
                <strong className={styles.selector__optionName}>
                  {pickupPoint.name}
                </strong>
                <span className={styles.selector__optionAddress}>
                  {pickupPoint.address}
                </span>
              </span>
            </button>
          ))}
        </div>
      )}

      {selectedId && (
        <div className={styles.selector__details}>
          {pickupPointQuery.isPending && <Loader text="Проверяем ПВЗ..." />}
          {pickupPointQuery.isError && (
            <p className={styles.selector__error}>
              Не удалось получить актуальную информацию о ПВЗ.
            </p>
          )}
          {pickupPointQuery.data && (
            <>
              <div className={styles.selector__detailsContent}>
                <span className={styles.selector__detailsLabel}>
                  Выбранный пункт
                </span>
                <strong className={styles.selector__detailsName}>
                  {pickupPointQuery.data.name}
                </strong>
                <span className={styles.selector__detailsAddress}>
                  {pickupPointQuery.data.address}
                </span>
              </div>
              <Button
                className={styles.selector__submit}
                disabled={pickupMutation.isPending}
                onClick={() => pickupMutation.mutate(selectedId)}
              >
                {pickupMutation.isPending
                  ? "Подтверждаем передачу..."
                  : "Отнести вещь в этот ПВЗ"}
              </Button>
            </>
          )}
        </div>
      )}

      {pickupMutation.isError && (
        <p className={styles.selector__error} role="alert">
          {getItemErrorMessage(
            pickupMutation.error,
            "Не удалось отметить передачу вещи в ПВЗ.",
          )}
        </p>
      )}
    </div>
  );
};
