import {
  memo,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import clsx from "clsx";
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
import type { MascotLevel } from "../../../Features/Mascot/mascot.types";
import { ProfileRatings } from "../../Widgets/ProfileRatings/ProfileRatings";
import { useMascot } from "../../../Hooks/useMascot";
import { Mascot } from "../../UI/Mascot/Mascot";

type TBlockAction = {
  type: "block" | "unblock";
  userId: string;
  nickname: string;
};

type TBlockedAssistantPosition = {
  height: number;
  left: number;
  ready: boolean;
  top: number;
  width: number;
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
  const { reactTo, reset, setGuideSuppressed } = useMascot();
  const [blockAction, setBlockAction] = useState<TBlockAction | null>(null);
  const [activeBlockedUserId, setActiveBlockedUserId] = useState<string | null>(
    null,
  );
  const blockedListRef = useRef<HTMLDivElement>(null);
  const blockedReactionActiveRef = useRef(false);
  const blockedUserRefs = useRef(new Map<string, HTMLDivElement>());
  const [blockedAssistantPosition, setBlockedAssistantPosition] =
    useState<TBlockedAssistantPosition>({
      height: 0,
      left: 0,
      ready: false,
      top: 0,
      width: 0,
    });
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
    blockedReactionActiveRef.current = false;
    setActiveBlockedUserId(null);
    reset();
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

  useEffect(() => {
    if (profileQuery.isError || blockedUsersQuery.isError) {
      reactTo("ERROR");
    }
  }, [
    blockedUsersQuery.isError,
    profileQuery.isError,
    reactTo,
  ]);

  useEffect(
    () => () => {
      setGuideSuppressed(false);
      if (blockedReactionActiveRef.current) {
        reset();
      }
    },
    [reset, setGuideSuppressed],
  );

  const moveBlockedAssistant = useCallback(
    (userId: string, rowElement?: HTMLDivElement | null) => {
      const listElement = blockedListRef.current;
      const blockedUserElement =
        rowElement ?? blockedUserRefs.current.get(userId);
      const slotElement =
        blockedUserElement?.querySelector<HTMLElement>(
          "[data-blocked-assistant-slot]",
        );

      if (!listElement || !slotElement) {
        return;
      }

      const listRect = listElement.getBoundingClientRect();
      const slotRect = slotElement.getBoundingClientRect();
      setBlockedAssistantPosition({
        height: Math.round(slotRect.height),
        left: Math.round(slotRect.left - listRect.left),
        ready: true,
        top: Math.round(slotRect.top - listRect.top),
        width: Math.round(slotRect.width),
      });
    },
    [],
  );

  useLayoutEffect(() => {
    if (!activeBlockedUserId) {
      return;
    }

    const listElement = blockedListRef.current;
    const blockedUserElement = blockedUserRefs.current.get(activeBlockedUserId);

    if (!listElement || !blockedUserElement) {
      return;
    }

    const updatePosition = () =>
      moveBlockedAssistant(activeBlockedUserId, blockedUserElement);
    const resizeObserver = new ResizeObserver(updatePosition);

    updatePosition();
    resizeObserver.observe(listElement);
    resizeObserver.observe(blockedUserElement);
    window.addEventListener("resize", updatePosition);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", updatePosition);
    };
  }, [activeBlockedUserId, moveBlockedAssistant]);

  const firstBlockedUserId = blockedUsers[0]?.id;
  useLayoutEffect(() => {
    if (
      !isOwnProfile ||
      blockedAssistantPosition.ready ||
      !firstBlockedUserId
    ) {
      return;
    }

    moveBlockedAssistant(firstBlockedUserId);
  }, [
    blockedAssistantPosition.ready,
    firstBlockedUserId,
    isOwnProfile,
    moveBlockedAssistant,
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
  const experience = user.experience;

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

  const activateBlockedUser = (
    userId: string,
    rowElement: HTMLDivElement,
  ) => {
    moveBlockedAssistant(userId, rowElement);

    if (activeBlockedUserId === userId) {
      return;
    }

    blockedReactionActiveRef.current = true;
    setActiveBlockedUserId(userId);
    reactTo("BLOCKED_USER_HOVERED");
  };

  const deactivateBlockedUser = (userId: string) => {
    if (activeBlockedUserId !== userId) {
      return;
    }

    blockedReactionActiveRef.current = false;
    setActiveBlockedUserId(null);
    reset();
  };

  const blockedAssistantStyle = {
    "--blocked-assistant-height": `${blockedAssistantPosition.height}px`,
    "--blocked-assistant-left": `${blockedAssistantPosition.left}px`,
    "--blocked-assistant-top": `${blockedAssistantPosition.top}px`,
    "--blocked-assistant-width": `${blockedAssistantPosition.width}px`,
  } as CSSProperties;

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

        <section className={styles.profile__experience}>
          <div className={styles.profile__experienceMascot}>
            <Mascot
              level={experience.level as MascotLevel}
              message={null}
              placement="inline"
              size="small"
            />
          </div>

          <div className={styles.profile__experienceContent}>
            <div className={styles.profile__experienceHeader}>
              <div>
                <span className={styles.profile__experienceEyebrow}>
                  Развитие Уми
                </span>
                <h2 className={styles.profile__experienceTitle}>
                  Уровень {experience.level} из {experience.max_level}
                </h2>
              </div>
              <strong className={styles.profile__experienceTotal}>
                {experience.total_xp} XP
              </strong>
            </div>

            <div
              aria-label={`Прогресс уровня: ${experience.progress_percent}%`}
              aria-valuemax={100}
              aria-valuemin={0}
              aria-valuenow={experience.progress_percent}
              className={styles.profile__experienceTrack}
              role="progressbar"
            >
              <span
                className={styles.profile__experienceProgress}
                style={{ width: `${experience.progress_percent}%` }}
              />
            </div>

            <p className={styles.profile__experienceDescription}>
              {experience.is_max_level
                ? "Максимальный уровень достигнут — продолжайте завершать обмены и накапливать опыт."
                : `До следующего уровня: ${experience.deals_to_next_level} ${experience.deals_to_next_level === 1 ? "завершённый обмен" : "завершённых обмена"}.`}
            </p>
          </div>
        </section>
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
        <section
          className={styles.profile__blocked}
          onMouseEnter={() => setGuideSuppressed(true)}
          onMouseLeave={(event) => {
            setGuideSuppressed(false);

            if (!event.currentTarget.contains(document.activeElement)) {
              blockedReactionActiveRef.current = false;
              setActiveBlockedUserId(null);
              reset();
            }
          }}
        >
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
                ref={blockedListRef}
              >
                {blockedUsers.map((blockedUser) => (
                  <div
                    className={styles.profile__blockedUser}
                    key={blockedUser.id}
                    ref={(element) => {
                      if (element) {
                        blockedUserRefs.current.set(blockedUser.id, element);
                      } else {
                        blockedUserRefs.current.delete(blockedUser.id);
                      }
                    }}
                    onBlur={(event) => {
                      if (
                        event.relatedTarget instanceof Node &&
                        event.currentTarget.contains(event.relatedTarget)
                      ) {
                        return;
                      }
                      deactivateBlockedUser(blockedUser.id);
                    }}
                    onFocus={(event) =>
                      activateBlockedUser(blockedUser.id, event.currentTarget)
                    }
                    onMouseEnter={(event) =>
                      activateBlockedUser(blockedUser.id, event.currentTarget)
                    }
                    onMouseLeave={(event) => {
                      if (event.currentTarget.contains(document.activeElement)) {
                        return;
                      }
                      deactivateBlockedUser(blockedUser.id);
                    }}
                  >
                    <Link
                      className={styles.profile__blockedIdentity}
                      to={`/profile/${blockedUser.id}`}
                      onClick={() => deactivateBlockedUser(blockedUser.id)}
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

                    <span
                      aria-hidden="true"
                      className={styles.profile__blockedAssistantSlot}
                      data-blocked-assistant-slot
                    />

                    <Button
                      className={styles.profile__blockedAction}
                      color="transparent"
                      disabled={blockMutationPending}
                      type="button"
                      onClick={() => {
                        deactivateBlockedUser(blockedUser.id);
                        setBlockAction({
                          type: "unblock",
                          userId: blockedUser.id,
                          nickname: blockedUser.nickname,
                        });
                      }}
                    >
                      Разблокировать
                    </Button>
                  </div>
                ))}
                <div
                  aria-hidden={!activeBlockedUserId}
                  aria-live="polite"
                  className={clsx(
                    styles.profile__blockedAssistant,
                    activeBlockedUserId &&
                      blockedAssistantPosition.ready &&
                      styles.profile__blockedAssistant_visible,
                  )}
                  style={blockedAssistantStyle}
                >
                  <Mascot
                    className={styles.profile__blockedAssistantMascot}
                    message="Плохой! Плохой!"
                    mode="attention"
                    mood="angry"
                    movement="kick"
                    placement="chat"
                    size="tiny"
                  />
                </div>
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
