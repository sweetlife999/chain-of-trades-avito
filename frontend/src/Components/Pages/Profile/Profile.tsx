import { memo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { CalendarDays, CircleCheck, CircleX, Star } from "lucide-react";

import styles from "./Styles.module.scss";

import {
  blockUser,
  getBlockedUsers,
  getUserById,
  unblockUser,
} from "../../../Api/auth/auth";
import { Button } from "../../UI/Button/Button";
import { ConfirmationPopup } from "../../UI/ConfirmationPopup/ConfirmationPopup";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { getAvatarGradient } from "../../Utils/getAvatarGradient";

const ProfileComponent = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id } = useParams();
  const { user: currentUser } = useAuthSelector();
  const [blockAction, setBlockAction] = useState<"block" | "unblock" | null>(null);
  const isOwnProfile = !id || id === currentUser?.id;

  const profileQuery = useQuery({
    queryKey: ["users", id],
    queryFn: () => getUserById(id ?? ""),
    enabled: Boolean(id && id !== currentUser?.id),
    retry: false,
  });

  const blockedUsersQuery = useQuery({
    queryKey: ["users", "blocks"],
    queryFn: getBlockedUsers,
    enabled: Boolean(currentUser && !isOwnProfile),
    retry: false,
  });

  const refreshAfterBlockChange = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["users", "blocks"] }),
      queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
    ]);
    setBlockAction(null);
  };

  const blockMutation = useMutation({
    mutationFn: (userId: string) => blockUser(userId),
    onSuccess: refreshAfterBlockChange,
  });

  const unblockMutation = useMutation({
    mutationFn: (userId: string) => unblockUser(userId),
    onSuccess: refreshAfterBlockChange,
  });

  const user = isOwnProfile ? currentUser : profileQuery.data;

  if (!isOwnProfile && profileQuery.isPending) {
    return <p className={styles.profile__empty}>Загрузка профиля...</p>;
  }

  if (!user || (!isOwnProfile && profileQuery.isError)) {
    return (
      <div className={styles.profile}>
        <p className={styles.profile__empty}>Пользователь не найден</p>
      </div>
    );
  }

  const createdAt = new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(new Date(user.created_at));

  const isBlocked =
    blockedUsersQuery.data?.some((blockedUser) => blockedUser.id === user.id) ??
    false;
  const blockMutationPending =
    blockMutation.isPending || unblockMutation.isPending;
  const blockMutationError =
    blockMutation.isError || unblockMutation.isError
      ? "Не удалось изменить блокировку. Повторите попытку."
      : undefined;

  const confirmBlockAction = () => {
    if (blockAction === "block") {
      blockMutation.mutate(user.id);
      return;
    }

    if (blockAction === "unblock") {
      unblockMutation.mutate(user.id);
    }
  };

  return (
    <div className={styles.profile}>
      <section className={styles.profile__card}>
        <div className={styles.profile__main}>
          <div
            className={styles.profile__avatar}
            style={{
              background: getAvatarGradient(user.id),
            }}
          >
            {user.photo_url ? (
              <img
                className={styles.profile__avatarImage}
                src={user.photo_url}
                alt={user.nickname}
              />
            ) : (
              <span className={styles.profile__avatarPlaceholder}>
                {user.nickname.charAt(0).toUpperCase()}
              </span>
            )}
          </div>

          <div className={styles.profile__information}>
            <p className={styles.profile__nickname}>{user.nickname}</p>

            {user.description && (
              <p className={styles.profile__description}>{user.description}</p>
            )}

            <div className={styles.profile__created}>
              <CalendarDays size={16} />
              <span>На сайте с {createdAt}</span>
            </div>
          </div>
        </div>

        {isOwnProfile && currentUser && (
          <Button
            color="transparent"
            type="button"
            onClick={() => navigate("/profile/edit")}
          >
            Редактировать
          </Button>
        )}

        {!isOwnProfile && currentUser && (
          <Button
            color={isBlocked ? "transparent" : "danger"}
            disabled={blockedUsersQuery.isPending || blockedUsersQuery.isError}
            type="button"
            onClick={() => setBlockAction(isBlocked ? "unblock" : "block")}
          >
            {blockedUsersQuery.isPending
              ? "Проверяем..."
              : blockedUsersQuery.isError
                ? "Не удалось проверить блокировку"
                : isBlocked
                  ? "Разблокировать"
                  : "Заблокировать"}
          </Button>
        )}

        <div className={styles.profile__statistics}>
          <div className={styles.profile__statistic}>
            <div className={styles.profile__statisticValue}>
              <span>{(user.rating ?? 0).toFixed(1)}</span>
              <Star
                className={styles.profile__ratingIcon}
                size={20}
                fill="currentColor"
              />
            </div>
            <span className={styles.profile__statisticLabel}>Рейтинг</span>
          </div>

          <div className={styles.profile__statistic}>
            <div className={styles.profile__statisticValue}>
              <CircleCheck
                className={styles.profile__completedIcon}
                size={20}
              />
              <span>{user.deals_completed}</span>
            </div>
            <span className={styles.profile__statisticLabel}>
              Завершено обменов
            </span>
          </div>

          <div className={styles.profile__statistic}>
            <div className={styles.profile__statisticValue}>
              <CircleX className={styles.profile__brokenIcon} size={20} />
              <span>{user.deals_broken}</span>
            </div>
            <span className={styles.profile__statisticLabel}>
              Сорвано обменов
            </span>
          </div>
        </div>
      </section>

      <section className={styles.profile__reviews}>
        <div className={styles.profile__reviewsHeader}>
          <h2 className={styles.profile__reviewsTitle}>Отзывы</h2>
          <span className={styles.profile__reviewsCount}>0</span>
        </div>

        <div className={styles.profile__reviewsEmpty}>
          <p className={styles.profile__reviewsEmptyTitle}>Отзывов пока нет</p>
          <p className={styles.profile__reviewsEmptyDescription}>
            После завершённых обменов здесь появятся отзывы других
            пользователей.
          </p>
        </div>
      </section>

      {blockAction && (
        <ConfirmationPopup
          confirmColor={blockAction === "block" ? "danger" : "green"}
          confirmLabel={
            blockAction === "block" ? "Заблокировать" : "Разблокировать"
          }
          description={
            blockAction === "block"
              ? "Пользователь не сможет попадать с вами в новые обмены. Общие неподтверждённые предложения будут отменены."
              : "Пользователь снова сможет попадать с вами в новые обмены. Существующие обмены не изменятся."
          }
          error={blockMutationError}
          isPending={blockMutationPending}
          pendingLabel={
            blockAction === "block" ? "Блокируем..." : "Разблокируем..."
          }
          title={
            blockAction === "block"
              ? `Заблокировать ${user.nickname}?`
              : `Разблокировать ${user.nickname}?`
          }
          onClose={() => {
            if (!blockMutationPending) {
              setBlockAction(null);
            }
          }}
          onConfirm={confirmBlockAction}
        />
      )}
    </div>
  );
};

export const Profile = memo(ProfileComponent);
