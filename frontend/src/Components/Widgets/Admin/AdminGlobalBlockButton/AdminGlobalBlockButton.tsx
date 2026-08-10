import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ShieldBan, ShieldCheck } from "lucide-react";

import {
  blockAdminUser,
  getAdminErrorMessage,
  unblockAdminUser,
} from "../../../../Api/admin/admin";
import { Button } from "../../../UI/Button/Button";
import { ConfirmationPopup } from "../../../UI/ConfirmationPopup/ConfirmationPopup";
import type { TAdminUserBlockResponse } from "../../../../Api/admin/admin.types";
import styles from "./Styles.module.scss";

type TAdminGlobalBlockButtonProps = {
  userId: string;
  nickname: string;
};

export const AdminGlobalBlockButton = ({
  userId,
  nickname,
}: TAdminGlobalBlockButtonProps) => {
  const queryClient = useQueryClient();
  const blockStateQueryKey = ["admin", "users", "global-block", userId] as const;
  const cachedBlockState =
    queryClient.getQueryData<TAdminUserBlockResponse>(blockStateQueryKey);
  const [isGloballyBlocked, setIsGloballyBlocked] = useState(
    cachedBlockState?.is_blocked ?? false,
  );
  const [popupOpen, setPopupOpen] = useState(false);

  const mutation = useMutation({
    mutationFn: () =>
      isGloballyBlocked ? unblockAdminUser(userId) : blockAdminUser(userId),
    onSuccess: async (response) => {
      queryClient.setQueryData(blockStateQueryKey, response);
      setIsGloballyBlocked(response.is_blocked);
      setPopupOpen(false);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["admin", "audit-log"] }),
        queryClient.invalidateQueries({ queryKey: ["admin", "dashboard"] }),
        queryClient.invalidateQueries({
          queryKey: ["admin", "users", "exchanges", userId],
        }),
      ]);
    },
  });

  const closePopup = () => {
    if (mutation.isPending) {
      return;
    }

    mutation.reset();
    setPopupOpen(false);
  };

  return (
    <>
      <Button
        className={styles.globalBlock__button}
        color={isGloballyBlocked ? "green" : "danger"}
        type="button"
        onClick={() => {
          mutation.reset();
          setPopupOpen(true);
        }}
      >
        {isGloballyBlocked ? (
          <ShieldCheck aria-hidden="true" size={17} />
        ) : (
          <ShieldBan aria-hidden="true" size={17} />
        )}
        {isGloballyBlocked
          ? "Разблокировать глобально"
          : "Заблокировать глобально"}
      </Button>

      {popupOpen && (
        <ConfirmationPopup
          confirmColor={isGloballyBlocked ? "green" : "danger"}
          confirmLabel={
            isGloballyBlocked
              ? "Разблокировать глобально"
              : "Заблокировать глобально"
          }
          description={
            isGloballyBlocked
              ? "Глобальная блокировка будет снята. Пользователь снова сможет входить и использовать сервис."
              : "Пользователь сразу потеряет доступ к сервису: новый вход и использование ранее выданного JWT будут запрещены. Личный список блокировок пользователей не изменится."
          }
          error={
            mutation.isError
              ? getAdminErrorMessage(
                  mutation.error,
                  isGloballyBlocked
                    ? "Не удалось снять глобальную блокировку."
                    : "Не удалось глобально заблокировать пользователя.",
                )
              : undefined
          }
          isPending={mutation.isPending}
          pendingLabel={isGloballyBlocked ? "Разблокируем..." : "Блокируем..."}
          title={
            isGloballyBlocked
              ? `Разблокировать ${nickname} глобально?`
              : `Заблокировать ${nickname} глобально?`
          }
          onClose={closePopup}
          onConfirm={() => mutation.mutate()}
        />
      )}
    </>
  );
};
