import { memo } from "react";
import { useQuery } from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../Api/exchanges/exchanges";
import { FetchStatus } from "../FetchStatus/FetchStatus";
import { Post } from "../Post/Post";

const PostsListComponent = () => {
  const exchangesQuery = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
  });

  // const exchanges = (exchangesQuery.data ?? []).filter(
  //   (exchange) =>
  //     exchange.status === "proposed" &&
  //     !exchange.participants.some((participant) => participant.user.id === user?.id),
  // );

  const exchanges = exchangesQuery.data
  return (
    <FetchStatus status={exchangesQuery.status}>
      {(exchanges?.length !== 0 && exchanges) ? (
        <ul className={styles.posts}>
          {exchanges.map((exchange) => (
            <li key={exchange.id}>
              <Post exchange={exchange} />
            </li>
          ))}
        </ul>
      ) : (
        <p>Предложений обмена пока нет</p>
      )}
    </FetchStatus>
  );
};

export const PostsList = memo(PostsListComponent);
