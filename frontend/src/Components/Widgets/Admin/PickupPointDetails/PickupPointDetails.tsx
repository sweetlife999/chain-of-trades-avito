import { useEffect, type MouseEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { CalendarDays, MapPin, Pencil, Trash2, X } from "lucide-react";

import { getAdminPickupPoint } from "../../../../Api/admin/admin";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import styles from "./Styles.module.scss";

type TProps = {
  pickupPointId: string;
  onClose: () => void;
  onDelete: () => void;
  onEdit: () => void;
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "long",
    timeStyle: "short",
  }).format(new Date(value));

export const PickupPointDetails = ({
  pickupPointId,
  onClose,
  onDelete,
  onEdit,
}: TProps) => {
  const pickupPointQuery = useQuery({
    queryKey: ["admin", "pickup-point", pickupPointId],
    queryFn: () => getAdminPickupPoint(pickupPointId),
    retry: false,
  });

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  };

  const pickupPoint = pickupPointQuery.data;

  return (
    <div className={styles.pointDetails__overlay} onMouseDown={handleOverlayClick}>
      <section
        aria-labelledby="pickup-point-details-title"
        aria-modal="true"
        className={styles.pointDetails}
        role="dialog"
      >
        <header className={styles.pointDetails__header}>
          <span className={styles.pointDetails__icon} aria-hidden="true">
            <MapPin className={styles.pointDetails__iconImage} />
          </span>
          <div className={styles.pointDetails__heading}>
            <span className={styles.pointDetails__eyebrow}>Пункт выдачи</span>
            <h2
              className={styles.pointDetails__title}
              id="pickup-point-details-title"
            >
              {pickupPoint?.name ?? "Информация о ПВЗ"}
            </h2>
          </div>
          <button
            aria-label="Закрыть карточку ПВЗ"
            className={styles.pointDetails__close}
            onClick={onClose}
            type="button"
          >
            <X aria-hidden="true" className={styles.pointDetails__closeIcon} />
          </button>
        </header>

        {pickupPointQuery.isPending && (
          <div className={styles.pointDetails__state}>
            <Loader text="Загружаем ПВЗ..." />
          </div>
        )}

        {pickupPointQuery.isError && (
          <div className={styles.pointDetails__state} role="alert">
            <strong className={styles.pointDetails__stateTitle}>
              Не удалось загрузить ПВЗ
            </strong>
            <p className={styles.pointDetails__stateDescription}>
              Пункт мог быть удалён или сервер временно недоступен.
            </p>
            <Button
              color="light"
              size="s"
              onClick={() => void pickupPointQuery.refetch()}
            >
              Повторить
            </Button>
          </div>
        )}

        {pickupPoint && (
          <>
            <div className={styles.pointDetails__content}>
              <div className={styles.pointDetails__address}>
                <MapPin
                  aria-hidden="true"
                  className={styles.pointDetails__contentIcon}
                />
                <div className={styles.pointDetails__contentText}>
                  <span className={styles.pointDetails__label}>Адрес</span>
                  <strong className={styles.pointDetails__value}>
                    {pickupPoint.address}
                  </strong>
                </div>
              </div>

              <div className={styles.pointDetails__dates}>
                <CalendarDays
                  aria-hidden="true"
                  className={styles.pointDetails__contentIcon}
                />
                <div className={styles.pointDetails__contentText}>
                  <span className={styles.pointDetails__label}>
                    Создан и обновлён
                  </span>
                  <span className={styles.pointDetails__date}>
                    Создан: {formatDate(pickupPoint.created_at)}
                  </span>
                  <span className={styles.pointDetails__date}>
                    Обновлён: {formatDate(pickupPoint.updated_at)}
                  </span>
                </div>
              </div>

              <div className={styles.pointDetails__identifier}>
                <span className={styles.pointDetails__label}>ID</span>
                <code className={styles.pointDetails__id}>{pickupPoint.id}</code>
              </div>
            </div>

            <div className={styles.pointDetails__actions}>
              <Button
                className={styles.pointDetails__action}
                color="transparent"
                onClick={onEdit}
              >
                <Pencil aria-hidden="true" size={17} />
                Редактировать
              </Button>
              <Button
                className={styles.pointDetails__action}
                color="danger"
                onClick={onDelete}
              >
                <Trash2 aria-hidden="true" size={17} />
                Удалить
              </Button>
            </div>
          </>
        )}
      </section>
    </div>
  );
};
