import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ChevronLeft,
  ChevronRight,
  PackageCheck,
  RefreshCw,
  Truck,
} from "lucide-react";

import { getAdminExchanges } from "../../../../Api/admin/admin";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import { MarkDeliveredButton } from "../MarkDeliveredButton/MarkDeliveredButton";
import styles from "./Styles.module.scss";

const PAGE_SIZE = 10;

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));

export const AdminExchangeDelivery = () => {
  const [page, setPage] = useState(0);
  const offset = page * PAGE_SIZE;

  const exchangesQuery = useQuery({
    queryKey: [
      "admin",
      "exchanges",
      { status: "delivering", limit: PAGE_SIZE, offset },
    ],
    queryFn: () =>
      getAdminExchanges({
        status: "delivering",
        limit: PAGE_SIZE,
        offset,
      }),
    placeholderData: (previousData) => previousData,
    retry: false,
  });

  const exchanges = exchangesQuery.data?.exchanges ?? [];
  const total = exchangesQuery.data?.pagination.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const rangeStart = total === 0 ? 0 : offset + 1;
  const rangeEnd = Math.min(offset + exchanges.length, total);

  return (
    <section
      aria-labelledby="admin-exchange-delivery-title"
      className={styles.delivery}
    >
      <div className={styles.delivery__sectionHeader}>
        <div className={styles.delivery__heading}>
          <span className={styles.delivery__icon} aria-hidden="true">
            <PackageCheck size={22} />
          </span>
          <div className={styles.delivery__headingContent}>
            <h2
              className={styles.delivery__title}
              id="admin-exchange-delivery-title"
            >
              Завершение доставки
            </h2>
            <p className={styles.delivery__description}>
              Обмены в статусе доставки. Подтверждайте завершение после того,
              как все вещи действительно доставлены в ПВЗ.
            </p>
          </div>
        </div>

        {!exchangesQuery.isPending && (
          <Button
            className={styles.delivery__refresh}
            color="transparent"
            disabled={exchangesQuery.isFetching}
            size="s"
            type="button"
            onClick={() => void exchangesQuery.refetch()}
          >
            <RefreshCw aria-hidden="true" size={16} />
            {exchangesQuery.isFetching ? "Обновляем..." : "Обновить"}
          </Button>
        )}
      </div>

      {exchangesQuery.isPending && (
        <div className={styles.delivery__state}>
          <Loader text="Загружаем обмены в доставке..." />
        </div>
      )}

      {exchangesQuery.isError && (
        <div className={styles.delivery__state} role="alert">
          <strong className={styles.delivery__stateTitle}>
            Не удалось загрузить очередь доставки
          </strong>
          <p className={styles.delivery__stateDescription}>
            Проверьте соединение и права администратора.
          </p>
          <Button
            color="light"
            size="s"
            type="button"
            onClick={() => void exchangesQuery.refetch()}
          >
            Повторить
          </Button>
        </div>
      )}

      {!exchangesQuery.isPending &&
        !exchangesQuery.isError &&
        exchanges.length === 0 && (
          <div className={styles.delivery__state}>
            <Truck
              aria-hidden="true"
              className={styles.delivery__stateIcon}
              size={34}
            />
            <strong className={styles.delivery__stateTitle}>
              Доставок для подтверждения нет
            </strong>
            <p className={styles.delivery__stateDescription}>
              Здесь появятся обмены со статусом delivering.
            </p>
          </div>
        )}

      {!exchangesQuery.isError && exchanges.length > 0 && (
        <div
          aria-busy={exchangesQuery.isFetching}
          className={styles.delivery__list}
        >
          {exchanges.map((exchange) => (
            <article className={styles.delivery__card} key={exchange.id}>
              <div className={styles.delivery__cardContent}>
                <div className={styles.delivery__cardTop}>
                  <div className={styles.delivery__exchangeIdentity}>
                    <span className={styles.delivery__exchangeLabel}>Обмен</span>
                    <code className={styles.delivery__exchangeId}>
                      {exchange.id}
                    </code>
                  </div>
                  <span className={styles.delivery__status}>Доставляется</span>
                </div>

                <div className={styles.delivery__participants}>
                  {exchange.participants.map((participant) => (
                    <div
                      className={styles.delivery__participant}
                      key={`${exchange.id}-${participant.position}-${participant.user.id}`}
                    >
                      <strong className={styles.delivery__participantName}>
                        {participant.user.nickname}
                      </strong>
                      <span className={styles.delivery__participantItem}>
                        {participant.gives_item.title}
                      </span>
                      <span className={styles.delivery__participantPickup}>
                        {participant.gives_item.pickup_point
                          ? `ПВЗ: ${participant.gives_item.pickup_point.name}`
                          : "ПВЗ не указан"}
                      </span>
                    </div>
                  ))}
                </div>

                <time
                  className={styles.delivery__date}
                  dateTime={exchange.updated_at}
                >
                  Обновлён {formatDate(exchange.updated_at)}
                </time>
              </div>

              <div className={styles.delivery__action}>
                <MarkDeliveredButton
                  exchangeId={exchange.id}
                  onDelivered={() => {
                    if (exchanges.length === 1 && page > 0) {
                      setPage((currentPage) => currentPage - 1);
                    }
                  }}
                />
              </div>
            </article>
          ))}
        </div>
      )}

      {!exchangesQuery.isError && total > PAGE_SIZE && (
        <nav aria-label="Пагинация доставок" className={styles.pagination}>
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
              <ChevronLeft aria-hidden="true" size={16} />
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
              <ChevronRight aria-hidden="true" size={16} />
            </Button>
          </div>
        </nav>
      )}
    </section>
  );
};
