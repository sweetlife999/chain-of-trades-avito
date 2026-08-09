import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw, UsersRound } from "lucide-react";

import styles from "./Styles.module.scss";
import { getAdminUserExchanges } from "../../../../Api/admin/admin";
import type { TAdminExchange } from "../../../../Api/admin/admin.types";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import { CancelExchangeButton } from "../CancelExchangeButton/CancelExchangeButton";
import { MarkDeliveredButton } from "../MarkDeliveredButton/MarkDeliveredButton";

const PAGE_SIZE = 5;

type TAdminUserExchangesProps = {
  userId: string;
  userNickname: string;
};

const statusLabels = {
  proposed: "Ожидает подтверждения",
  confirmed: "Подтверждён",
  delivering: "Доставляется",
  delivered: "Ожидает получения",
  completed: "Завершён",
  cancelled: "Отменён",
} as const;

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));

const getExchangeTitle = (exchange: TAdminExchange, userId: string) => {
  const participant =
    exchange.participants.find(({ user }) => user.id === userId) ??
    exchange.participants[0];

  return participant
    ? `${participant.gives_item.title} → ${participant.receives_item.title}`
    : "Обмен без участников";
};

export const AdminUserExchanges = ({
  userId,
  userNickname,
}: TAdminUserExchangesProps) => {
  const [page, setPage] = useState(0);
  const offset = page * PAGE_SIZE;

  const exchangesQuery = useQuery({
    queryKey: [
      "admin",
      "users",
      "exchanges",
      userId,
      { limit: PAGE_SIZE, offset },
    ],
    queryFn: () =>
      getAdminUserExchanges(userId, { limit: PAGE_SIZE, offset }),
    placeholderData: (previousData) => previousData,
    retry: false,
  });

  const total = exchangesQuery.data?.pagination.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const exchanges = exchangesQuery.data?.exchanges ?? [];
  const rangeStart = total === 0 ? 0 : offset + 1;
  const rangeEnd = Math.min(offset + exchanges.length, total);

  return (
    <section
      aria-labelledby="admin-user-exchanges-title"
      className={styles.exchanges}
    >
      <header className={styles.exchanges__header}>
        <div className={styles.exchanges__heading}>
          <span className={styles.exchanges__eyebrow}>Доступ администратора</span>
          <h2 className={styles.exchanges__title} id="admin-user-exchanges-title">
            Активные обмены пользователя
          </h2>
          <p className={styles.exchanges__description}>
            Цепочки пользователя {userNickname}. Всего: {total}
          </p>
        </div>

        <Button
          className={styles.exchanges__refresh}
          color="transparent"
          disabled={exchangesQuery.isFetching}
          size="s"
          type="button"
          onClick={() => {
            void exchangesQuery.refetch();
          }}
        >
          <RefreshCw
            aria-hidden="true"
            className={styles.exchanges__refreshIcon}
            size={17}
          />
          Обновить
        </Button>
      </header>

      {exchangesQuery.isPending && (
        <div className={styles.exchanges__state}>
          <Loader text="Загружаем обмены пользователя..." />
        </div>
      )}

      {exchangesQuery.isError && (
        <div className={styles.exchanges__state} role="alert">
          <h3 className={styles.exchanges__stateTitle}>
            Не удалось загрузить обмены
          </h3>
          <p className={styles.exchanges__stateDescription}>
            Проверьте соединение или права администратора и повторите запрос.
          </p>
          <Button
            className={styles.exchanges__stateAction}
            color="light"
            size="s"
            type="button"
            onClick={() => {
              void exchangesQuery.refetch();
            }}
          >
            Повторить
          </Button>
        </div>
      )}

      {!exchangesQuery.isPending &&
        !exchangesQuery.isError &&
        exchanges.length === 0 && (
          <div className={styles.exchanges__state}>
            <UsersRound
              aria-hidden="true"
              className={styles.exchanges__stateIcon}
              size={34}
            />
            <h3 className={styles.exchanges__stateTitle}>
              Активных обменов нет
            </h3>
            <p className={styles.exchanges__stateDescription}>
              У пользователя нет доступных для администрирования цепочек.
            </p>
          </div>
        )}

      {!exchangesQuery.isError && exchanges.length > 0 && (
        <div
          aria-busy={exchangesQuery.isFetching}
          className={styles.exchanges__list}
        >
          {exchanges.map((exchange) => (
            <article className={styles.exchanges__card} key={exchange.id}>
              <div className={styles.exchanges__information}>
                <div className={styles.exchanges__titleRow}>
                  <h3 className={styles.exchanges__cardTitle}>
                    {getExchangeTitle(exchange, userId)}
                  </h3>
                  <span
                    className={`${styles.exchanges__status} ${styles[`exchanges__status_${exchange.status}`]}`}
                  >
                    {statusLabels[exchange.status]}
                  </span>
                </div>

                <div className={styles.exchanges__meta}>
                  <span className={styles.exchanges__metaItem}>
                    {exchange.participants.length} участника
                  </span>
                  <time
                    className={styles.exchanges__metaItem}
                    dateTime={exchange.created_at}
                  >
                    Создан {formatDate(exchange.created_at)}
                  </time>
                </div>

                <div className={styles.exchanges__participants}>
                  {exchange.participants.map((participant) => (
                    <span
                      className={styles.exchanges__participant}
                      key={participant.user.id}
                    >
                      <strong className={styles.exchanges__participantName}>
                        {participant.user.nickname}
                      </strong>
                      <small className={styles.exchanges__participantItem}>
                        {participant.gives_item.title}
                      </small>
                    </span>
                  ))}
                </div>
              </div>

              <div className={styles.exchanges__actions}>
                {exchange.status === "delivering" && (
                  <MarkDeliveredButton exchangeId={exchange.id} />
                )}
                {["proposed", "confirmed"].includes(exchange.status) && (
                  <CancelExchangeButton
                    exchangeId={exchange.id}
                    onCancelled={() => {
                      if (exchanges.length === 1 && page > 0) {
                        setPage((currentPage) => currentPage - 1);
                      }
                    }}
                  />
                )}
              </div>
            </article>
          ))}
        </div>
      )}

      {!exchangesQuery.isError && total > PAGE_SIZE && (
        <nav aria-label="Пагинация обменов" className={styles.pagination}>
          <span className={styles.pagination__summary}>
            Показано {rangeStart}–{rangeEnd} из {total}
          </span>
          <div className={styles.pagination__controls}>
            <Button
              color="transparent"
              disabled={page === 0 || exchangesQuery.isFetching}
              size="s"
              type="button"
              onClick={() => setPage((currentPage) => currentPage - 1)}
            >
              Назад
            </Button>
            <span className={styles.pagination__current}>
              {page + 1} / {totalPages}
            </span>
            <Button
              color="transparent"
              disabled={page + 1 >= totalPages || exchangesQuery.isFetching}
              size="s"
              type="button"
              onClick={() => setPage((currentPage) => currentPage + 1)}
            >
              Далее
            </Button>
          </div>
        </nav>
      )}
    </section>
  );
};
