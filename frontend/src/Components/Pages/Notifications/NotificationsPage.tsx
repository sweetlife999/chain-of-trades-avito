import { memo, useMemo, useState } from "react";
import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import clsx from "clsx";
import {
  Bell,
  CheckCheck,
  CheckCircle2,
  Headset,
  MessageCircle,
  PackageCheck,
  Sparkles,
  Truck,
  XCircle,
} from "lucide-react";
import { Link, useNavigate } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  getNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from "../../../Api/notifications/notifications";
import type {
  TNotification,
  TNotificationKind,
} from "../../../Api/notifications/notifications.types";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { Button } from "../../UI/Button/Button";
import {
  formatNotificationTime,
  getNotificationDescription,
  getNotificationPath,
  getNotificationTitle,
} from "../../Widgets/Notifications/notificationPresentation";

type TFilter = "all" | "unread";

const pageSize = 30;

const iconByKind = (kind: TNotificationKind) => {
  switch (kind) {
    case "exchange_proposed":
      return Sparkles;
    case "text":
      return MessageCircle;
    case "support_message":
      return Headset;
    case "exchange_delivering":
      return Truck;
    case "exchange_delivered":
    case "participant_delivered_item":
      return PackageCheck;
    case "exchange_superseded":
    case "exchange_item_withdrawn":
    case "participant_declined":
      return XCircle;
    default:
      return CheckCircle2;
  }
};

const NotificationsPageComponent = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { isAdmin, isAuth } = useAuthSelector();
  const [filter, setFilter] = useState<TFilter>("all");
  const unreadOnly = filter === "unread";

  const notificationsQuery = useInfiniteQuery({
    queryKey: ["notifications", { unreadOnly }],
    queryFn: ({ pageParam }) =>
      getNotifications({ unreadOnly, limit: pageSize, offset: pageParam }),
    initialPageParam: 0,
    enabled: isAuth,
    getNextPageParam: (lastPage) =>
      lastPage.notifications.length < lastPage.limit
        ? undefined
        : lastPage.offset + lastPage.limit,
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
  });

  const notifications = useMemo(
    () =>
      notificationsQuery.data?.pages.flatMap(
        (page) => page.notifications,
      ) ?? [],
    [notificationsQuery.data],
  );
  const unreadCount = notificationsQuery.data?.pages[0]?.unread_count ?? 0;

  const invalidateNotifications = () =>
    queryClient.invalidateQueries({ queryKey: ["notifications"] });

  const markReadMutation = useMutation({
    mutationFn: markNotificationRead,
    onSuccess: invalidateNotifications,
  });
  const markAllMutation = useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: invalidateNotifications,
  });

  const openNotification = (notification: TNotification) => {
    if (!notification.is_read) {
      markReadMutation.mutate(notification.id);
    }
    navigate(getNotificationPath(notification, isAdmin));
  };

  if (!isAuth) {
    return (
      <section className={styles.page}>
        <div className={styles.page__empty}>
          <Bell aria-hidden="true" size={34} strokeWidth={1.6} />
          <h1 className={styles.page__emptyTitle}>Войдите в аккаунт</h1>
          <p className={styles.page__emptyDescription}>
            Уведомления доступны только авторизованным пользователям.
          </p>
          <Link className={styles.page__login} to="/login">
            Войти
          </Link>
        </div>
      </section>
    );
  }

  return (
    <section className={styles.page}>
      <div className={styles.page__header}>
        <div className={styles.page__heading}>
          <h1 className={styles.page__title}>Уведомления</h1>
          <p className={styles.page__description}>
            Новые цепочки, сообщения и изменения статусов обмена.
          </p>
        </div>
        <Button
          className={styles.page__markAll}
          color="light"
          disabled={unreadCount === 0 || markAllMutation.isPending}
          onClick={() => markAllMutation.mutate()}
        >
          <CheckCheck aria-hidden="true" size={18} />
          {markAllMutation.isPending ? "Отмечаем..." : "Прочитать все"}
        </Button>
      </div>

      <div className={styles.page__toolbar}>
        <div className={styles.page__filters} aria-label="Фильтр уведомлений">
          <button
            className={clsx(
              styles.page__filter,
              filter === "all" && styles.page__filter_active,
            )}
            type="button"
            onClick={() => setFilter("all")}
          >
            Все
          </button>
          <button
            className={clsx(
              styles.page__filter,
              filter === "unread" && styles.page__filter_active,
            )}
            type="button"
            onClick={() => setFilter("unread")}
          >
            Непрочитанные
            {unreadCount > 0 && (
              <span className={styles.page__filterCount}>{unreadCount}</span>
            )}
          </button>
        </div>
      </div>

      {notificationsQuery.isPending && (
        <p className={styles.page__state}>Загружаем уведомления...</p>
      )}
      {notificationsQuery.isError && (
        <div className={styles.page__state_error}>
          <p>Не удалось загрузить уведомления.</p>
          <Button color="light" onClick={() => notificationsQuery.refetch()}>
            Повторить
          </Button>
        </div>
      )}

      {!notificationsQuery.isPending &&
        !notificationsQuery.isError &&
        notifications.length === 0 && (
          <div className={styles.page__empty}>
            <Bell aria-hidden="true" size={34} strokeWidth={1.6} />
            <h2 className={styles.page__emptyTitle}>
              {unreadOnly
                ? "Все уведомления прочитаны"
                : "Уведомлений пока нет"}
            </h2>
            <p className={styles.page__emptyDescription}>
              Здесь появятся найденные цепочки, сообщения и этапы обмена.
            </p>
          </div>
        )}

      {notifications.length > 0 && (
        <div className={styles.page__list}>
          {notifications.map((notification) => {
            const Icon = iconByKind(notification.kind);

            return (
              <button
                className={clsx(
                  styles.page__item,
                  !notification.is_read && styles.page__item_unread,
                )}
                key={notification.id}
                type="button"
                onClick={() => openNotification(notification)}
              >
                <span className={styles.page__icon}>
                  <Icon aria-hidden="true" size={21} strokeWidth={1.8} />
                </span>
                <span className={styles.page__content}>
                  <span className={styles.page__itemHeader}>
                    <strong className={styles.page__itemTitle}>
                      {getNotificationTitle(notification)}
                    </strong>
                    <time
                      className={styles.page__time}
                      dateTime={notification.created_at}
                    >
                      {formatNotificationTime(notification.created_at)}
                    </time>
                  </span>
                  <span className={styles.page__itemDescription}>
                    {getNotificationDescription(notification)}
                  </span>
                </span>
                {!notification.is_read && (
                  <span
                    aria-label="Непрочитанное уведомление"
                    className={styles.page__unreadDot}
                  />
                )}
              </button>
            );
          })}
        </div>
      )}

      {notificationsQuery.hasNextPage && (
        <Button
          centered
          className={styles.page__loadMore}
          color="light"
          disabled={notificationsQuery.isFetchingNextPage}
          onClick={() => notificationsQuery.fetchNextPage()}
        >
          {notificationsQuery.isFetchingNextPage
            ? "Загружаем..."
            : "Показать ещё"}
        </Button>
      )}
    </section>
  );
};

export const NotificationsPage = memo(NotificationsPageComponent);
