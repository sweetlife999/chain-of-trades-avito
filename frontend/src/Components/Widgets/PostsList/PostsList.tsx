import { memo } from "react";
import { useQuery } from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../Api/exchanges/exchanges";
import { getItems } from "../../../Api/items/items";
import { FetchStatus } from "../FetchStatus/FetchStatus";
import { Post } from "../Post/Post";
import { ExchangeSearchStatus } from "../ExchangeSearchStatus/ExchangeSearchStatus";

const PostsListComponent = () => {
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

  const exchanges = exchangesQuery.data ?? [];

  return (
    <section className={styles.postsSection}>
      <ExchangeSearchStatus
        exchangesCount={exchanges.length}
        isLoading={exchangesQuery.isPending || exchangesQuery.isFetching}
        hasOwnItems={hasOwnItems}
      />

      <FetchStatus status={exchangesQuery.status}>
        {exchanges.length ? (
          <ul className={styles.posts}>
            {exchanges.map((exchange) => (
              <li className={styles.posts__item} key={exchange.id}>
                <Post exchange={exchange} />
              </li>
            ))}
          </ul>
        ) : (
          <div className={styles.postsSection__empty}>
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
