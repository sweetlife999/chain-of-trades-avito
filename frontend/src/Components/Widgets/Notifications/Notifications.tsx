import { memo, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Bell } from "lucide-react";
import { Link } from "react-router-dom";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../Api/exchanges/exchanges";
import type { TExchange } from "../../../Api/exchanges/exchanges.types";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";

const getExchangeTitle = (exchange: TExchange, userId: string) => {
  const participant =
    exchange.participants.find(({ user }) => user.id === userId) ??
    exchange.participants[0];

  return participant
    ? `${participant.gives_item.title} → ${participant.receives_item.title}`
    : "Цепочка обмена";
};

const getUnreadLabel = (count: number) => {
  const lastTwoDigits = count % 100;
  const lastDigit = count % 10;

  if (lastTwoDigits >= 11 && lastTwoDigits <= 14) {
    return `${count} новых сообщений`;
  }

  if (lastDigit === 1) {
    return `${count} новое сообщение`;
  }

  if (lastDigit >= 2 && lastDigit <= 4) {
    return `${count} новых сообщения`;
  }

  return `${count} новых сообщений`;
};

const NotificationsComponent = () => {
  const { isAuth, user } = useAuthSelector();
  const [isOpen, setIsOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  const exchangesQuery = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
    enabled: Boolean(isAuth && user),
    refetchInterval: 3000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    retry: false,
  });

  const unreadExchanges = useMemo(() => {
    if (!user) {
      return [];
    }

    return (exchangesQuery.data ?? [])
      .filter(
        (exchange) =>
          exchange.unread_count > 0 &&
          exchange.participants.some(
            (participant) => participant.user.id === user.id,
          ),
      )
      .sort(
        (first, second) =>
          new Date(second.updated_at).getTime() -
          new Date(first.updated_at).getTime(),
      );
  }, [exchangesQuery.data, user]);

  const unreadCount = unreadExchanges.reduce(
    (total, exchange) => total + exchange.unread_count,
    0,
  );
  const badge = unreadCount > 99 ? "99+" : String(unreadCount);

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

  if (!isAuth || !user) {
    return null;
  }

  return (
    <div className={styles.notifications} ref={rootRef}>
      <button
        aria-expanded={isOpen}
        aria-label={
          unreadCount > 0
            ? `Уведомления: ${unreadCount} непрочитанных`
            : "Уведомления"
        }
        className={styles.notifications__trigger}
        type="button"
        onClick={() => setIsOpen((open) => !open)}
      >
        <Bell aria-hidden="true" size={23} strokeWidth={1.9} />
        {unreadCount > 0 && (
          <span className={styles.notifications__badge}>{badge}</span>
        )}
      </button>

      {isOpen && (
        <div className={styles.notifications__panel}>
          <div className={styles.notifications__header}>
            <div>
              <h2>Уведомления</h2>
              <p>Новые сообщения в ваших цепочках</p>
            </div>
            {unreadCount > 0 && (
              <span className={styles.notifications__total}>{badge}</span>
            )}
          </div>

          {exchangesQuery.isPending && (
            <p className={styles.notifications__state}>Загрузка...</p>
          )}

          {exchangesQuery.isError && (
            <p className={styles.notifications__state}>
              Не удалось загрузить уведомления.
            </p>
          )}

          {!exchangesQuery.isPending &&
            !exchangesQuery.isError &&
            unreadExchanges.length === 0 && (
              <div className={styles.notifications__empty}>
                <Bell aria-hidden="true" size={28} strokeWidth={1.6} />
                <strong>Новых уведомлений нет</strong>
                <span>Здесь появятся непрочитанные сообщения из цепочек.</span>
              </div>
            )}

          {!exchangesQuery.isPending &&
            !exchangesQuery.isError &&
            unreadExchanges.length > 0 && (
              <div className={styles.notifications__list}>
                {unreadExchanges.map((exchange) => (
                  <Link
                    className={styles.notifications__item}
                    key={exchange.id}
                    to={`/exchanges/${exchange.id}`}
                    onClick={() => setIsOpen(false)}
                  >
                    <span className={styles.notifications__itemContent}>
                      <strong>{getExchangeTitle(exchange, user.id)}</strong>
                      <span>{getUnreadLabel(exchange.unread_count)}</span>
                    </span>
                    <span className={styles.notifications__itemCount}>
                      {exchange.unread_count > 99
                        ? "99+"
                        : exchange.unread_count}
                    </span>
                  </Link>
                ))}
              </div>
            )}
        </div>
      )}
    </div>
  );
};

export const Notifications = memo(NotificationsComponent);
