import { memo } from "react";
import { Link } from "react-router-dom";

import styles from "./Styles.module.scss";
import type {
  TExchange,
  TExchangeStatus,
} from "../../../Api/exchanges/exchanges.types";

type TProps = {
  exchange: TExchange;
};

const statusLabels: Record<TExchangeStatus, string> = {
  proposed: "Предложение обмена",
  confirmed: "Обмен подтверждён",
  completed: "Обмен завершён",
  cancelled: "Обмен отменён",
};

const getTitle = (exchange: TExchange) => {
  const participant = exchange.participants[0];

  return participant
    ? `${participant.gives_item.title} → ${participant.receives_item.title}`
    : "Обмен без участников";
};

const PostComponent = ({ exchange }: TProps) => (
  <Link className={styles.post} to={`/exchanges/${exchange.id}`}>
    <div className={styles.post__top}>
      <span className={`${styles.post__status} ${styles[`post__status_${exchange.status}`]}`}>
        {statusLabels[exchange.status]}
      </span>
      <span>{exchange.participants.length} участников</span>
    </div>

    <h2>{getTitle(exchange)}</h2>

    <div className={styles.post__participants}>
      {exchange.participants.slice(0, 4).map(({ user }) => (
        <span key={user.id} title={user.nickname}>
          {user.nickname.charAt(0).toUpperCase()}
        </span>
      ))}
    </div>

    <span className={styles.post__open}>Открыть обмен</span>
  </Link>
);

export const Post = memo(PostComponent);
