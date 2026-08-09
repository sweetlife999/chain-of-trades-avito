import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MapPin, Pencil, Plus, Trash2, X } from "lucide-react";
import { useState } from "react";

import {
  deleteAdminPickupPoint,
  getAdminErrorMessage,
  getAdminPickupPoints,
} from "../../../../Api/admin/admin";
import type { TPickupPoint } from "../../../../Api/admin/admin.types";
import { Button } from "../../../UI/Button/Button";
import { ConfirmationPopup } from "../../../UI/ConfirmationPopup/ConfirmationPopup";
import { Loader } from "../../../UI/Loader/Loader";
import {
  PickupPointForm,
  type TPickupPointFormMode,
} from "../PickupPointForm/PickupPointForm";
import { PickupPointDetails } from "../PickupPointDetails/PickupPointDetails";
import styles from "./Styles.module.scss";

type TFormState = {
  mode: TPickupPointFormMode;
  pickupPointId?: string;
};

const formatDate = (date: string) =>
  new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium" }).format(
    new Date(date),
  );

export const PickupPoints = () => {
  const [form, setForm] = useState<TFormState | null>(null);
  const [pickupPointToDelete, setPickupPointToDelete] =
    useState<TPickupPoint | null>(null);
  const [selectedPickupPoint, setSelectedPickupPoint] =
    useState<TPickupPoint | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const pickupPointsQuery = useQuery({
    queryKey: ["admin", "pickup-points"],
    queryFn: getAdminPickupPoints,
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAdminPickupPoint,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ["admin", "pickup-points"],
        }),
        queryClient.invalidateQueries({ queryKey: ["admin", "dashboard"] }),
      ]);

      setPickupPointToDelete(null);
      setNotice("ПВЗ удалён");
    },
  });

  const openForm = (mode: TPickupPointFormMode, pickupPointId?: string) => {
    setNotice(null);
    setForm({ mode, pickupPointId });
  };

  return (
    <section className={styles.points} aria-labelledby="pickup-points-title">
      <div className={styles.admin__sectionHeader}>
        <div className={styles.admin__sectionHeading}>
          <h2 className={styles.admin__sectionTitle} id="pickup-points-title">
            Пункты выдачи
          </h2>
          <p className={styles.admin__sectionDescription}>
            {pickupPointsQuery.data
              ? `Всего пунктов: ${pickupPointsQuery.data.length}`
              : "Управление пунктами выдачи"}
          </p>
        </div>
        <Button
          className={styles.admin__createButton}
          onClick={() => openForm("create")}
        >
          <Plus aria-hidden="true" className={styles.admin__createIcon} />
          Создать ПВЗ
        </Button>
      </div>

      {notice && (
        <div className={styles.points__notice} role="status">
          <span className={styles.points__noticeText}>{notice}</span>
          <button
            className={styles.points__noticeClose}
            type="button"
            aria-label="Скрыть уведомление"
            onClick={() => setNotice(null)}
          >
            <X aria-hidden="true" className={styles.points__noticeCloseIcon} />
          </button>
        </div>
      )}

      {pickupPointsQuery.isPending && (
        <div className={styles.admin__state}>
          <Loader text="Загружаем пункты выдачи..." />
        </div>
      )}

      {pickupPointsQuery.isError && (
        <div className={styles.admin__state} role="alert">
          <h3 className={styles.admin__stateTitle}>Не удалось загрузить ПВЗ</h3>
          <p className={styles.admin__stateDescription}>
            Проверьте соединение с сервером и повторите запрос.
          </p>
          <Button
            className={styles.admin__stateAction}
            color="light"
            size="s"
            onClick={() => {
              void pickupPointsQuery.refetch();
            }}
          >
            Повторить
          </Button>
        </div>
      )}

      {pickupPointsQuery.data && pickupPointsQuery.data.length === 0 && (
        <div className={styles.admin__state}>
          <span className={styles.points__emptyIcon} aria-hidden="true">
            <MapPin className={styles.points__emptyIconImage} />
          </span>
          <h3 className={styles.admin__stateTitle}>Пунктов выдачи пока нет</h3>
          <p className={styles.admin__stateDescription}>
            Создайте первый ПВЗ, чтобы он появился в списке.
          </p>
        </div>
      )}

      {pickupPointsQuery.data && pickupPointsQuery.data.length > 0 && (
        <div className={styles.points__tableWrapper}>
          <table className={styles.points__table}>
            <thead className={styles.points__tableHead}>
              <tr className={styles.points__tableHeadRow}>
                <th className={`${styles.points__tableHeading} ${styles.points__tableHeading_name}`}>ПВЗ</th>
                <th className={`${styles.points__tableHeading} ${styles.points__tableHeading_address}`}>Адрес</th>
                <th className={`${styles.points__tableHeading} ${styles.points__tableHeading_date}`}>Создан</th>
                <th className={`${styles.points__tableHeading} ${styles.points__tableHeading_date}`}>Обновлён</th>
                <th className={`${styles.points__tableHeading} ${styles.points__tableHeading_actions}`}>
                  <span className={styles.admin__visuallyHidden}>Действия</span>
                </th>
              </tr>
            </thead>
            <tbody className={styles.points__tableBody}>
              {pickupPointsQuery.data.map((pickupPoint) => (
                <tr className={styles.points__tableRow} key={pickupPoint.id}>
                  <td className={styles.points__tableCell} data-label="ПВЗ">
                    <button
                      aria-label={`Открыть информацию о ${pickupPoint.name}`}
                      className={styles.points__detailsButton}
                      onClick={() => setSelectedPickupPoint(pickupPoint)}
                      type="button"
                    >
                      <span className={styles.points__nameIcon} aria-hidden="true">
                        <MapPin className={styles.points__nameIconImage} />
                      </span>
                      <strong className={styles.points__nameText}>
                        {pickupPoint.name}
                      </strong>
                    </button>
                  </td>
                  <td className={styles.points__tableCell} data-label="Адрес">
                    {pickupPoint.address}
                  </td>
                  <td className={`${styles.points__tableCell} ${styles.points__tableCell_date}`} data-label="Создан">
                    {formatDate(pickupPoint.created_at)}
                  </td>
                  <td className={`${styles.points__tableCell} ${styles.points__tableCell_date}`} data-label="Обновлён">
                    {formatDate(pickupPoint.updated_at)}
                  </td>
                  <td className={styles.points__tableCell} data-label="Действия">
                    <div className={styles.points__actions}>
                      <button
                        className={styles.points__actionButton}
                        type="button"
                        title="Редактировать"
                        aria-label={`Редактировать ${pickupPoint.name}`}
                        onClick={() => openForm("edit", pickupPoint.id)}
                      >
                        <Pencil aria-hidden="true" className={styles.points__actionIcon} />
                      </button>
                      <button
                        className={`${styles.points__actionButton} ${styles.points__actionButton_delete}`}
                        type="button"
                        title="Удалить"
                        aria-label={`Удалить ${pickupPoint.name}`}
                        onClick={() => {
                          deleteMutation.reset();
                          setPickupPointToDelete(pickupPoint);
                        }}
                      >
                        <Trash2 aria-hidden="true" className={styles.points__actionIcon} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {form && (
        <PickupPointForm
          key={`${form.mode}-${form.pickupPointId ?? "new"}`}
          mode={form.mode}
          pickupPointId={form.pickupPointId}
          onClose={() => setForm(null)}
          onSaved={(message) => {
            setForm(null);
            setNotice(message);
          }}
        />
      )}

      {selectedPickupPoint && (
        <PickupPointDetails
          pickupPointId={selectedPickupPoint.id}
          onClose={() => setSelectedPickupPoint(null)}
          onDelete={() => {
            deleteMutation.reset();
            setPickupPointToDelete(selectedPickupPoint);
            setSelectedPickupPoint(null);
          }}
          onEdit={() => {
            openForm("edit", selectedPickupPoint.id);
            setSelectedPickupPoint(null);
          }}
        />
      )}

      {pickupPointToDelete && (
        <ConfirmationPopup
          title="Удалить ПВЗ?"
          description={`Пункт «${pickupPointToDelete.name}» будет удалён без возможности восстановления. ПВЗ, связанный с объявлениями, удалить нельзя.`}
          confirmLabel="Удалить"
          pendingLabel="Удаляем..."
          isPending={deleteMutation.isPending}
          error={
            deleteMutation.isError
              ? getAdminErrorMessage(
                  deleteMutation.error,
                  "Не удалось удалить ПВЗ",
                )
              : undefined
          }
          onClose={() => {
            if (!deleteMutation.isPending) {
              setPickupPointToDelete(null);
              deleteMutation.reset();
            }
          }}
          onConfirm={() => deleteMutation.mutate(pickupPointToDelete.id)}
        />
      )}
    </section>
  );
};
