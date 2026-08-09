import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { PackageCheck } from "lucide-react";

import styles from "./Styles.module.scss";
import {
  getAdminErrorMessage,
  markAdminExchangeDelivered,
} from "../../../../Api/admin/admin";
import type { TExchange } from "../../../../Api/exchanges/exchanges.types";
import { Button } from "../../../UI/Button/Button";
import { ConfirmationPopup } from "../../../UI/ConfirmationPopup/ConfirmationPopup";

type TMarkDeliveredButtonProps = {
  exchangeId: string;
  onDelivered?: () => void;
};

export const MarkDeliveredButton = ({
  exchangeId,
  onDelivered,
}: TMarkDeliveredButtonProps) => {
  const [popupOpen, setPopupOpen] = useState(false);
  const queryClient = useQueryClient();

  const deliveredMutation = useMutation({
    mutationFn: () => markAdminExchangeDelivered(exchangeId),
    onSuccess: async () => {
      queryClient.setQueryData<TExchange>(
        ["exchanges", exchangeId],
        (exchange) =>
          exchange ? { ...exchange, status: "delivered" } : exchange,
      );

      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ["admin", "users", "exchanges"],
        }),
        queryClient.invalidateQueries({ queryKey: ["admin", "dashboard"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
      ]);

      setPopupOpen(false);
      onDelivered?.();
    },
  });

  const closePopup = () => {
    if (deliveredMutation.isPending) {
      return;
    }

    deliveredMutation.reset();
    setPopupOpen(false);
  };

  return (
    <>
      <Button
        className={styles.delivered__button}
        color="green"
        size="s"
        type="button"
        onClick={() => {
          deliveredMutation.reset();
          setPopupOpen(true);
        }}
      >
        <PackageCheck
          aria-hidden="true"
          className={styles.delivered__icon}
          size={17}
        />
        Завершить доставку
      </Button>

      {popupOpen && (
        <ConfirmationPopup
          confirmColor="green"
          confirmLabel="Перевести к получению"
          description="Обмен перейдёт из статуса доставки в статус получения. После этого участники смогут подтвердить получение своих вещей."
          error={
            deliveredMutation.isError
              ? getAdminErrorMessage(
                  deliveredMutation.error,
                  "Не удалось завершить доставку. Убедитесь, что обмен находится в доставке и все вещи уже переданы в ПВЗ.",
                )
              : undefined
          }
          isPending={deliveredMutation.isPending}
          pendingLabel="Завершаем доставку..."
          title="Доставка всех вещей завершена?"
          onClose={closePopup}
          onConfirm={() => deliveredMutation.mutate()}
        />
      )}
    </>
  );
};
