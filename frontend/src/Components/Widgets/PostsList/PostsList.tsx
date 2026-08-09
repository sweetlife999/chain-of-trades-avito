import { memo } from "react";
import { useQuery } from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../Api/exchanges/exchanges";
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

  // const exchanges = (exchangesQuery.data ?? []).filter(
  //   (exchange) =>
  //     exchange.status === "proposed" &&
  //     !exchange.participants.some((participant) => participant.user.id === user?.id),
  // );

  const exchanges = exchangesQuery.data ?? [];

  return (
    <section className={styles.postsSection}>
      <ExchangeSearchStatus
        exchangesCount={exchanges.length}
        isLoading={exchangesQuery.isPending || exchangesQuery.isFetching}
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
              Поиск продолжает работать. Добавьте вещь или вернитесь немного
              позже — новые варианты появятся автоматически.
            </p>
          </div>
        )}
      </FetchStatus>
    </section>
  );
};

export const PostsList = memo(PostsListComponent);
