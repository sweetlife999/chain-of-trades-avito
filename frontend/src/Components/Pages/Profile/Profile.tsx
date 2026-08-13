import { memo, useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";
import { CalendarDays, CircleCheck, CircleX } from "lucide-react";

import styles from "./Styles.module.scss";

import {
  blockUser,
  getBlockedUsers,
  getUserById,
  unblockUser,
} from "../../../Api/auth/auth";
import { Button } from "../../UI/Button/Button";
import { ConfirmationPopup } from "../../UI/ConfirmationPopup/ConfirmationPopup";
import { Rating } from "../../UI/Rating/Rating";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { getAvatarGradient } from "../../Utils/getAvatarGradient";
import { AdminGlobalBlockButton } from "../../Widgets/Admin/AdminGlobalBlockButton/AdminGlobalBlockButton";
import { AdminUserExchanges } from "../../Widgets/Admin/AdminUserExchanges/AdminUserExchanges";
import { ProfileRatings } from "../../Widgets/ProfileRatings/ProfileRatings";
import { useMascot } from "../../../Hooks/useMascot";

type TBlockAction = {
  type: "block" | "unblock";
  userId: string;
  nickname: string;
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(new Date(value));

const ProfileComponent = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id } = useParams();
  const { user: currentUser } = useAuthSelector();
  const { reactTo } = useMascot();
  const [blockAction, setBlockAction] = useState<TBlockAction | null>(null);
  const isOwnProfile = !id || id === currentUser?.id;
  const profileUserId = id ?? currentUser?.id;

  const profileQuery = useQuery({
    queryKey: ["users", profileUserId],
    queryFn: () => getUserById(profileUserId ?? ""),
    enabled: Boolean(profileUserId),
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
    retry: false,
  });

  const blockedUsersQuery = useQuery({
    queryKey: ["users", "blocks"],
    queryFn: getBlockedUsers,
    enabled: Boolean(currentUser),
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

  const user = profileQuery.data ?? (isOwnProfile ? currentUser : undefined);
  const blockedUsers = blockedUsersQuery.data ?? [];
  const blockedUserIds = blockedUsers.map(({ id: userId }) => userId).join(":");

  useEffect(() => {
    if (
      !isOwnProfile ||
      !blockedUsersQuery.isSuccess ||
      blockedUserIds.length === 0
    ) {
      return;
    }

    const timerId = window.setTimeout(
      () => reactTo("BLOCKED_USERS_VIEWED"),
      4300,
    );

    return () => window.clearTimeout(timerId);
  }, [
    blockedUserIds,
    blockedUsersQuery.isSuccess,
    isOwnProfile,
    reactTo,
  ]);

  useEffect(() => {
    if (profileQuery.isError || blockedUsersQuery.isError) {
      reactTo("ERROR");
    }
  }, [
    blockedUsersQuery.isError,
    profileQuery.isError,
    reactTo,
  ]);

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

  const createdAt = formatDate(user.created_at);
  const isBlocked = blockedUsers.some((blockedUser) => blockedUser.id === user.id);
  const blockMutationPending = blockMutation.isPending || unblockMutation.isPending;
  const blockMutationError =
    blockMutation.isError || unblockMutation.isError
      ? "Не удалось изменить блокировку. Повторите попытку."
      : undefined;

  const confirmBlockAction = () => {
    if (!blockAction) {
      return;
    }

    if (blockAction.type === "block") {
      blockMutation.mutate(blockAction.userId);
      return;
    }

    unblockMutation.mutate(blockAction.userId);
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
            <h1 className={styles.profile__nickname}>{user.nickname}</h1>

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
          <div className={styles.profile__actions}>
            <Button
              className={styles.profile__action}
              color="transparent"
              type="button"
              onClick={() => navigate("/profile/edit")}
            >
              Редактировать
            </Button>
          </div>
        )}

        {!isOwnProfile && currentUser && (
          <div className={styles.profile__actions}>
            <Button
              className={styles.profile__action}
              color={isBlocked ? "transparent" : "danger"}
              disabled={blockedUsersQuery.isPending || blockedUsersQuery.isError}
              type="button"
              onClick={() =>
                setBlockAction({
                  type: isBlocked ? "unblock" : "block",
                  userId: user.id,
                  nickname: user.nickname,
                })
              }
            >
              {blockedUsersQuery.isPending
                ? "Проверяем..."
                : blockedUsersQuery.isError
                  ? "Не удалось проверить блокировку"
                  : isBlocked
                    ? "Разблокировать"
                    : "Заблокировать"}
            </Button>

            {currentUser.is_admin && (
              <AdminGlobalBlockButton
                nickname={user.nickname}
                userId={user.id}
              />
            )}
          </div>
        )}

        <div className={styles.profile__statistics}>
          <div className={styles.profile__statistic}>
            <div className={styles.profile__statisticValue}>
              <Rating count={user.ratings_count} value={user.rating} />
            </div>
            <span className={styles.profile__statisticLabel}>Рейтинг</span>
          </div>

          <div className={styles.profile__statistic}>
            <div className={styles.profile__statisticValue}>
              <CircleCheck className={styles.profile__completedIcon} size={20} />
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
            <span className={styles.profile__statisticLabel}>Сорвано обменов</span>
          </div>
        </div>
      </section>

      <ProfileRatings
        key={user.id}
        ratingsCount={user.ratings_count}
        userId={user.id}
      />

      {!isOwnProfile && currentUser?.is_admin && (
        <AdminUserExchanges
          userId={user.id}
          userNickname={user.nickname}
        />
      )}

      {isOwnProfile && currentUser && (
        <section className={styles.profile__blocked}>
          <div className={styles.profile__blockedHeader}>
            <h2 className={styles.profile__blockedTitle}>
              Заблокированные пользователи
            </h2>
            <span className={styles.profile__blockedCount}>
              {blockedUsers.length}
            </span>
          </div>

          {blockedUsersQuery.isPending && (
            <p className={styles.profile__blockedState}>
              Загружаем список блокировок...
            </p>
          )}

          {blockedUsersQuery.isError && (
            <p className={styles.profile__blockedState}>
              Не удалось загрузить список блокировок.
            </p>
          )}

          {!blockedUsersQuery.isPending &&
            !blockedUsersQuery.isError &&
            blockedUsers.length === 0 && (
              <div className={styles.profile__blockedEmpty}>
                <p className={styles.profile__blockedEmptyTitle}>
                  Список блокировок пуст
                </p>
                <p className={styles.profile__blockedEmptyDescription}>
                  Заблокированные пользователи будут отображаться здесь.
                </p>
              </div>
            )}

          {!blockedUsersQuery.isPending &&
            !blockedUsersQuery.isError &&
            blockedUsers.length > 0 && (
              <div
                className={styles.profile__blockedList}
                data-mascot-anchor="blocked-users"
              >
                {blockedUsers.map((blockedUser) => (
                  <div
                    className={styles.profile__blockedUser}
                    data-mascot-kick-target
                    key={blockedUser.id}
                  >
                    <Link
                      className={styles.profile__blockedIdentity}
                      to={`/profile/${blockedUser.id}`}
                    >
                      <span
                        className={styles.profile__blockedAvatar}
                        style={{ background: getAvatarGradient(blockedUser.id) }}
                      >
                        {blockedUser.photo_url ? (
                          <img
                            alt={blockedUser.nickname}
                            className={styles.profile__blockedAvatarImage}
                            src={blockedUser.photo_url}
                          />
                        ) : (
                          blockedUser.nickname.charAt(0).toUpperCase()
                        )}
                      </span>

                      <span className={styles.profile__blockedInformation}>
                        <strong className={styles.profile__blockedName}>
                          {blockedUser.nickname}
                        </strong>
                        <span className={styles.profile__blockedDate}>
                          Заблокирован {formatDate(blockedUser.blocked_at)}
                        </span>
                      </span>
                    </Link>

                    <Button
                      className={styles.profile__blockedAction}
                      color="transparent"
                      disabled={blockMutationPending}
                      type="button"
                      onClick={() =>
                        setBlockAction({
                          type: "unblock",
                          userId: blockedUser.id,
                          nickname: blockedUser.nickname,
                        })
                      }
                    >
                      Разблокировать
                    </Button>
                  </div>
                ))}
              </div>
            )}
        </section>
      )}

      {blockAction && (
        <ConfirmationPopup
          confirmColor={blockAction.type === "block" ? "danger" : "green"}
          confirmLabel={
            blockAction.type === "block" ? "Заблокировать" : "Разблокировать"
          }
          description={
            blockAction.type === "block"
              ? "Пользователь не сможет попадать с вами в новые обмены. Общие неподтверждённые предложения будут отменены."
              : "Пользователь снова сможет попадать с вами в новые обмены. Существующие обмены не изменятся."
          }
          error={blockMutationError}
          isPending={blockMutationPending}
          pendingLabel={
            blockAction.type === "block" ? "Блокируем..." : "Разблокируем..."
          }
          title={
            blockAction.type === "block"
              ? `Заблокировать ${blockAction.nickname}?`
              : `Разблокировать ${blockAction.nickname}?`
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
