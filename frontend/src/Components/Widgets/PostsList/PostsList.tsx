import { memo, useEffect, useMemo, useRef } from "react";
import { useQuery } from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../Api/exchanges/exchanges";
import { getItems } from "../../../Api/items/items";
import { FetchStatus } from "../FetchStatus/FetchStatus";
import { Post } from "../Post/Post";
import { ExchangeSearchStatus } from "../ExchangeSearchStatus/ExchangeSearchStatus";
import { useMascot } from "../../../Hooks/useMascot";

const PostsListComponent = () => {
  const { reactTo } = useMascot();
  const previousExchangeIdsRef = useRef<Set<string> | null>(null);
  const exchangesQuery = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  });

  const { data: hasOwnItems = false } = useQuery({
    queryKey: ["items"],
    queryFn: getItems,
    select: (items) => items.length > 0,
  });

  const exchanges = useMemo(
    () => exchangesQuery.data ?? [],
    [exchangesQuery.data],
  );
  const exchangeIds = exchanges.map(({ id }) => id).join(":");

  useEffect(() => {
    if (!exchangesQuery.isSuccess) {
      return;
    }

    const currentIds = new Set(exchanges.map(({ id }) => id));
    const previousIds = previousExchangeIdsRef.current;
    previousExchangeIdsRef.current = currentIds;

    if (previousIds === null) {
      if (exchanges.length === 0) {
        reactTo("EMPTY_EXCHANGES");
      }
      return;
    }

    const hasNewExchange = exchanges.some(({ id }) => !previousIds.has(id));

    if (hasNewExchange) {
      reactTo("CHAIN_AVAILABLE");
      return;
    }

    if (exchanges.length === 0) {
      reactTo("EMPTY_EXCHANGES");
    }
  }, [exchangeIds, exchanges, exchangesQuery.isSuccess, reactTo]);

  useEffect(() => {
    if (exchangesQuery.isError) {
      reactTo("ERROR");
    }
  }, [exchangesQuery.isError, reactTo]);

  return (
    <section className={styles.postsSection}>
      <ExchangeSearchStatus
        exchangesCount={exchanges.length}
        isLoading={exchangesQuery.isPending || exchangesQuery.isFetching}
        hasOwnItems={hasOwnItems}
      />

      <FetchStatus status={exchangesQuery.status}>
        {exchanges.length ? (
          <ul className={styles.posts} data-mascot-anchor="chains-list">
            {exchanges.map((exchange) => (
              <li className={styles.posts__item} key={exchange.id}>
                <Post exchange={exchange} />
              </li>
            ))}
          </ul>
        ) : (
          <div
            className={styles.postsSection__empty}
            data-mascot-anchor="exchanges-empty"
          >
            <strong className={styles.postsSection__emptyTitle}>
              Предложений пока нет
            </strong>

            <p className={styles.postsSection__emptyDescription}>
              {hasOwnItems
                ? "Поиск продолжает работать. Вернитесь немного позже — новые варианты появятся автоматически."
                : "Добавьте вещь, чтобы начать поиск подходящих вариантов обмена."}
            </p>
          </div>
        )}
      </FetchStatus>
    </section>
  );
};

export const PostsList = memo(PostsListComponent);
