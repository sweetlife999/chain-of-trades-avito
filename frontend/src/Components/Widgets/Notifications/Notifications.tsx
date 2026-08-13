import { memo, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, CheckCheck } from "lucide-react";
import { Link, useLocation } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  getNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from "../../../Api/notifications/notifications";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { useMascot } from "../../../Hooks/useMascot";
import { shouldShowMascotGuide } from "../../../Features/Mascot/mascotVisibility";
import { Mascot } from "../../UI/Mascot/Mascot";
import {
  formatNotificationTime,
  getNotificationDescription,
  getNotificationPath,
  getNotificationTitle,
} from "./notificationPresentation";

const previewLimit = 6;

const NotificationsComponent = () => {
  const { pathname } = useLocation();
  const { isAdmin, isAuth } = useAuthSelector();
  const { reactTo, reset } = useMascot();
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const wasOpenRef = useRef(false);
  const previousNotificationIdsRef = useRef<Set<string> | null>(null);

  const notificationsQuery = useQuery({
    queryKey: ["notifications", "preview"],
    queryFn: () =>
      getNotifications({ unreadOnly: true, limit: previewLimit, offset: 0 }),
    enabled: isAuth,
    refetchInterval: 3000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    retry: false,
  });

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

  const notifications = notificationsQuery.data?.notifications ?? [];
  const unreadCount = notificationsQuery.data?.unread_count ?? 0;
  const badge = unreadCount > 99 ? "99+" : String(unreadCount);
  const notificationIds = notifications.map(({ id }) => id).join("|");

  useEffect(() => {
    if (!notificationsQuery.isSuccess) {
      return;
    }

    const currentIds = new Set(notificationIds.split("|").filter(Boolean));
    const previousIds = previousNotificationIdsRef.current;
    const hasNewNotification =
      previousIds !== null &&
      [...currentIds].some((notificationId) => !previousIds.has(notificationId));

    previousNotificationIdsRef.current = currentIds;

    if (
      hasNewNotification &&
      !isOpen &&
      shouldShowMascotGuide(pathname)
    ) {
      reactTo("NEW_NOTIFICATION");
    }
  }, [
    isOpen,
    notificationIds,
    notificationsQuery.isSuccess,
    pathname,
    reactTo,
  ]);

  useEffect(() => {
    if (isOpen && shouldShowMascotGuide(pathname)) {
      reactTo("NOTIFICATIONS_PREVIEW_OPENED");
    } else if (wasOpenRef.current) {
      reset();
    }

    wasOpenRef.current = isOpen;
  }, [isOpen, pathname, reactTo, reset]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (
        rootRef.current &&
        event.target instanceof Node &&
        !rootRef.current.contains(event.target)
      ) {
        setIsOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsOpen(false);
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  if (!isAuth) {
    return null;
  }

  return (
    <div className={styles.notifications} ref={rootRef}>
      <button
        aria-expanded={isOpen}
        aria-haspopup="dialog"
        aria-label={
          unreadCount > 0
            ? `Уведомления: ${unreadCount} непрочитанных`
            : "Уведомления"
        }
        className={styles.notifications__trigger}
        data-mascot-anchor="notifications-trigger"
        data-tutorial-target="notifications"
        type="button"
        onClick={() => setIsOpen((open) => !open)}
      >
        <Bell
          aria-hidden="true"
          className={styles.notifications__triggerIcon}
          size={23}
          strokeWidth={1.9}
        />
        {unreadCount > 0 && (
          <span className={styles.notifications__badge}>{badge}</span>
        )}
      </button>

      {isOpen && (
        <div
          aria-label="Непрочитанные уведомления"
          className={styles.notifications__panel}
          data-mascot-anchor="notifications-preview"
          role="dialog"
        >
          <div className={styles.notifications__header}>
            <div className={styles.notifications__heading}>
              <h2 className={styles.notifications__title}>Уведомления</h2>
              <p className={styles.notifications__description}>
                Цепочки, сообщения и этапы обмена
              </p>
            </div>
            {unreadCount > 0 && (
              <button
                aria-label="Отметить все уведомления прочитанными"
                className={styles.notifications__markAll}
                disabled={markAllMutation.isPending}
                title="Прочитать все"
                type="button"
                onClick={() => markAllMutation.mutate()}
              >
                <CheckCheck aria-hidden="true" size={18} />
              </button>
            )}
          </div>

          {shouldShowMascotGuide(pathname) && (
            <div
              className={styles.notifications__assistant}
              data-mascot-protected
            >
              <Mascot
                className={styles.notifications__assistantMascot}
                message="Смотрю, что нового…"
                mode="attention"
                mood="reading"
                movement="scan"
                placement="chat"
                size="tiny"
              />
            </div>
          )}

          {notificationsQuery.isPending && (
            <p className={styles.notifications__state}>Загрузка...</p>
          )}

          {notificationsQuery.isError && (
            <div className={styles.notifications__state}>
              <p>Не удалось загрузить уведомления.</p>
              <button
                className={styles.notifications__retry}
                type="button"
                onClick={() => notificationsQuery.refetch()}
              >
                Повторить
              </button>
            </div>
          )}

          {!notificationsQuery.isPending &&
            !notificationsQuery.isError &&
            notifications.length === 0 && (
              <div className={styles.notifications__empty}>
                <Bell
                  aria-hidden="true"
                  className={styles.notifications__emptyIcon}
                  size={28}
                  strokeWidth={1.6}
                />
                <strong className={styles.notifications__emptyTitle}>
                  Новых уведомлений нет
                </strong>
                <span className={styles.notifications__emptyDescription}>
                  Здесь появятся новые цепочки, сообщения и этапы обмена.
                </span>
              </div>
            )}

          {!notificationsQuery.isPending &&
            !notificationsQuery.isError &&
            notifications.length > 0 && (
              <div className={styles.notifications__list}>
                {notifications.map((notification) => (
                  <Link
                    className={styles.notifications__item}
                    key={notification.id}
                    to={getNotificationPath(notification, isAdmin)}
                    onClick={() => {
                      markReadMutation.mutate(notification.id);
                      setIsOpen(false);
                    }}
                  >
                    <span className={styles.notifications__unreadDot} />
                    <span className={styles.notifications__itemContent}>
                      <strong className={styles.notifications__itemTitle}>
                        {getNotificationTitle(notification)}
                      </strong>
                      <span className={styles.notifications__itemMeta}>
                        {getNotificationDescription(notification)}
                      </span>
                      <time
                        className={styles.notifications__itemTime}
                        dateTime={notification.created_at}
                      >
                        {formatNotificationTime(notification.created_at)}
                      </time>
                    </span>
                  </Link>
                ))}
              </div>
            )}

          <Link
            className={styles.notifications__footer}
            to="/notifications"
            onClick={() => setIsOpen(false)}
          >
            Все уведомления
            {unreadCount > 0 && (
              <span className={styles.notifications__footerCount}>{badge}</span>
            )}
          </Link>
        </div>
      )}
    </div>
  );
};

export const Notifications = memo(NotificationsComponent);
