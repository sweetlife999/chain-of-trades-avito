import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Ban } from "lucide-react";

import styles from "./Styles.module.scss";
import {
  cancelAdminExchange,
  getAdminErrorMessage,
} from "../../../../Api/admin/admin";
import type { TExchange } from "../../../../Api/exchanges/exchanges.types";
import { Button } from "../../../UI/Button/Button";
import { ConfirmationPopup } from "../../../UI/ConfirmationPopup/ConfirmationPopup";

type TCancelExchangeButtonProps = {
  exchangeId: string;
  onCancelled?: () => void;
};

export const CancelExchangeButton = ({
  exchangeId,
  onCancelled,
}: TCancelExchangeButtonProps) => {
  const [popupOpen, setPopupOpen] = useState(false);
  const queryClient = useQueryClient();

  const cancelMutation = useMutation({
    mutationFn: () => cancelAdminExchange(exchangeId),
    onSuccess: async () => {
      queryClient.setQueryData<TExchange>(
        ["exchanges", exchangeId],
        (exchange) =>
          exchange ? { ...exchange, status: "cancelled" } : exchange,
      );

      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ["admin", "users", "exchanges"],
        }),
        queryClient.invalidateQueries({ queryKey: ["admin", "dashboard"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
        queryClient.invalidateQueries({ queryKey: ["items"] }),
      ]);

      setPopupOpen(false);
      onCancelled?.();
    },
  });

  const closePopup = () => {
    if (cancelMutation.isPending) {
      return;
    }

    cancelMutation.reset();
    setPopupOpen(false);
  };

  return (
    <>
      <Button
        className={styles.cancel__button}
        color="danger"
        size="s"
        type="button"
        onClick={() => {
          cancelMutation.reset();
          setPopupOpen(true);
        }}
      >
        <Ban
          aria-hidden="true"
          className={styles.cancel__icon}
          size={17}
        />
        Отменить обмен
      </Button>

      {popupOpen && (
        <ConfirmationPopup
          confirmLabel="Отменить обмен"
          description="Обмен будет принудительно отменён. Зарезервированные объявления снова станут доступны, а для подтверждённой цепочки запустится повторный поиск."
          error={
            cancelMutation.isError
              ? getAdminErrorMessage(
                  cancelMutation.error,
                  "Не удалось отменить обмен. Повторите попытку.",
                )
              : undefined
          }
          isPending={cancelMutation.isPending}
          pendingLabel="Отменяем..."
          title="Принудительно отменить обмен?"
          onClose={closePopup}
          onConfirm={() => cancelMutation.mutate()}
        />
      )}
    </>
  );
};
