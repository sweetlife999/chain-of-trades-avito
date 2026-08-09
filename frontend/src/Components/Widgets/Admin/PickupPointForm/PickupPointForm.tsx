import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MapPin, X } from "lucide-react";
import { useEffect, type MouseEvent } from "react";
import { useForm } from "react-hook-form";

import {
  createAdminPickupPoint,
  getAdminErrorMessage,
  getAdminPickupPoint,
  updateAdminPickupPoint,
} from "../../../../Api/admin/admin";
import {
  CreatePickupPointSchema,
  type TCreatePickupPoint,
  type TPickupPoint,
  type TUpdatePickupPoint,
} from "../../../../Api/admin/admin.types";
import { Button } from "../../../UI/Button/Button";
import { Input } from "../../../UI/Input/Input";
import { Loader } from "../../../UI/Loader/Loader";
import styles from "./Styles.module.scss";

export type TPickupPointFormMode = "create" | "edit";

type TProps = {
  mode: TPickupPointFormMode;
  pickupPointId?: string;
  onClose: () => void;
  onSaved: (message: string) => void;
};

export const PickupPointForm = ({
  mode,
  pickupPointId,
  onClose,
  onSaved,
}: TProps) => {
  const queryClient = useQueryClient();
  const pickupPointQuery = useQuery({
    queryKey: ["admin", "pickup-point", pickupPointId],
    queryFn: () => getAdminPickupPoint(pickupPointId ?? ""),
    enabled: mode === "edit" && Boolean(pickupPointId),
  });

  const {
    register,
    handleSubmit,
    reset,
    setError,
    clearErrors,
    formState: { errors },
  } = useForm<TCreatePickupPoint>({
    resolver: zodResolver(CreatePickupPointSchema),
    defaultValues: { name: "", address: "" },
  });

  useEffect(() => {
    if (pickupPointQuery.data) {
      reset({
        name: pickupPointQuery.data.name,
        address: pickupPointQuery.data.address,
      });
    }
  }, [pickupPointQuery.data, reset]);

  const saveMutation = useMutation<
    TPickupPoint,
    unknown,
    TCreatePickupPoint | TUpdatePickupPoint
  >({
    mutationFn: (request) => {
      if (mode === "create") {
        return createAdminPickupPoint(CreatePickupPointSchema.parse(request));
      }

      return updateAdminPickupPoint(pickupPointId ?? "", request);
    },
    onSuccess: async (pickupPoint) => {
      queryClient.setQueryData(
        ["admin", "pickup-point", pickupPoint.id],
        pickupPoint,
      );

      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ["admin", "pickup-points"],
        }),
        queryClient.invalidateQueries({ queryKey: ["admin", "dashboard"] }),
      ]);

      onSaved(mode === "create" ? "ПВЗ создан" : "ПВЗ обновлён");
    },
    onError: (error) => {
      setError("root.server", {
        type: "server",
        message: getAdminErrorMessage(
          error,
          mode === "create"
            ? "Не удалось создать ПВЗ"
            : "Не удалось изменить ПВЗ",
        ),
      });
    },
  });

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !saveMutation.isPending) {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose, saveMutation.isPending]);

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget && !saveMutation.isPending) {
      onClose();
    }
  };

  const onSubmit = (formData: TCreatePickupPoint) => {
    clearErrors("root.server");

    if (mode === "create") {
      saveMutation.mutate(formData);
      return;
    }

    const currentPickupPoint = pickupPointQuery.data;

    if (!currentPickupPoint) {
      return;
    }

    const changes: TUpdatePickupPoint = {};

    if (formData.name !== currentPickupPoint.name) {
      changes.name = formData.name;
    }

    if (formData.address !== currentPickupPoint.address) {
      changes.address = formData.address;
    }

    if (Object.keys(changes).length === 0) {
      setError("root.server", {
        type: "manual",
        message: "Измените название или адрес перед сохранением",
      });
      return;
    }

    saveMutation.mutate(changes);
  };

  const isLoading = mode === "edit" && pickupPointQuery.isPending;
  const isError = mode === "edit" && pickupPointQuery.isError;

  return (
    <div className={styles.modal__overlay} onMouseDown={handleOverlayClick}>
      <section
        className={styles.modal}
        role="dialog"
        aria-modal="true"
        aria-labelledby="pickup-point-form-title"
      >
        <div className={styles.modal__header}>
          <span className={styles.modal__icon} aria-hidden="true">
            <MapPin className={styles.modal__iconImage} />
          </span>
          <div className={styles.modal__heading}>
            <h2 className={styles.modal__title} id="pickup-point-form-title">
              {mode === "create" ? "Создать ПВЗ" : "Редактировать ПВЗ"}
            </h2>
            <p className={styles.modal__description}>
              Укажите название и полный адрес пункта выдачи
            </p>
          </div>
          <button
            className={styles.modal__close}
            type="button"
            aria-label="Закрыть"
            disabled={saveMutation.isPending}
            onClick={onClose}
          >
            <X aria-hidden="true" className={styles.modal__closeIcon} />
          </button>
        </div>

        {isLoading && (
          <div className={styles.modal__state}>
            <Loader text="Загружаем ПВЗ..." />
          </div>
        )}

        {isError && (
          <div className={styles.modal__state} role="alert">
            <h3 className={styles.modal__stateTitle}>
              Не удалось загрузить ПВЗ
            </h3>
            <p className={styles.modal__stateDescription}>
              Пункт выдачи мог быть удалён или сервер временно недоступен.
            </p>
            <Button
              className={styles.modal__stateAction}
              color="light"
              size="s"
              onClick={() => {
                void pickupPointQuery.refetch();
              }}
            >
              Повторить
            </Button>
          </div>
        )}

        {!isLoading && !isError && (
          <form
            className={styles.modal__form}
            noValidate
            onSubmit={handleSubmit(onSubmit)}
          >
            <Input
              label="Название ПВЗ"
              placeholder="Например, ПВЗ на Ленина"
              required
              disabled={saveMutation.isPending}
              error={errors.name?.message}
              {...register("name")}
            />
            <Input
              label="Адрес"
              placeholder="Город, улица, дом"
              required
              disabled={saveMutation.isPending}
              error={errors.address?.message}
              {...register("address")}
            />

            {errors.root?.server && (
              <p className={styles.modal__error} role="alert">
                {errors.root.server.message}
              </p>
            )}

            <div className={styles.modal__actions}>
              <Button
                className={styles.modal__action}
                color="transparent"
                type="button"
                disabled={saveMutation.isPending}
                onClick={onClose}
              >
                Отмена
              </Button>
              <Button
                className={styles.modal__action}
                type="submit"
                disabled={saveMutation.isPending}
              >
                {saveMutation.isPending
                  ? "Сохраняем..."
                  : mode === "create"
                    ? "Создать"
                    : "Сохранить"}
              </Button>
            </div>
          </form>
        )}
      </section>
    </div>
  );
};
