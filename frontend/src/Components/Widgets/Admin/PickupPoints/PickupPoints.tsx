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
        <div>
          <h2 id="pickup-points-title">Пункты выдачи</h2>
          <p>
            {pickupPointsQuery.data
              ? `Всего пунктов: ${pickupPointsQuery.data.length}`
              : "Управление пунктами выдачи"}
          </p>
        </div>
        <Button onClick={() => openForm("create")}>
          <Plus aria-hidden="true" />
          Создать ПВЗ
        </Button>
      </div>

      {notice && (
        <div className={styles.points__notice} role="status">
          <span>{notice}</span>
          <button
            type="button"
            aria-label="Скрыть уведомление"
            onClick={() => setNotice(null)}
          >
            <X aria-hidden="true" />
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
          <h3>Не удалось загрузить ПВЗ</h3>
          <p>Проверьте соединение с сервером и повторите запрос.</p>
          <Button
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
            <MapPin />
          </span>
          <h3>Пунктов выдачи пока нет</h3>
          <p>Создайте первый ПВЗ, чтобы он появился в списке.</p>
        </div>
      )}

      {pickupPointsQuery.data && pickupPointsQuery.data.length > 0 && (
        <div className={styles.points__tableWrapper}>
          <table className={styles.points__table}>
            <thead>
              <tr>
                <th>ПВЗ</th>
                <th>Адрес</th>
                <th>Создан</th>
                <th>Обновлён</th>
                <th><span className={styles.admin__visuallyHidden}>Действия</span></th>
              </tr>
            </thead>
            <tbody>
              {pickupPointsQuery.data.map((pickupPoint) => (
                <tr key={pickupPoint.id}>
                  <td data-label="ПВЗ">
                    <span className={styles.points__name}>
                      <span aria-hidden="true"><MapPin /></span>
                      <strong>{pickupPoint.name}</strong>
                    </span>
                  </td>
                  <td data-label="Адрес">{pickupPoint.address}</td>
                  <td data-label="Создан">{formatDate(pickupPoint.created_at)}</td>
                  <td data-label="Обновлён">{formatDate(pickupPoint.updated_at)}</td>
                  <td data-label="Действия">
                    <div className={styles.points__actions}>
                      <button
                        type="button"
                        title="Редактировать"
                        aria-label={`Редактировать ${pickupPoint.name}`}
                        onClick={() => openForm("edit", pickupPoint.id)}
                      >
                        <Pencil aria-hidden="true" />
                      </button>
                      <button
                        className={styles.points__deleteButton}
                        type="button"
                        title="Удалить"
                        aria-label={`Удалить ${pickupPoint.name}`}
                        onClick={() => {
                          deleteMutation.reset();
                          setPickupPointToDelete(pickupPoint);
                        }}
                      >
                        <Trash2 aria-hidden="true" />
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
