import { memo, type KeyboardEvent } from "react";
import { Link, useNavigate } from "react-router-dom";

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
  delivering: "Вещи доставляются",
  delivered: "Вещь ожидает в ПВЗ",
  completed: "Обмен завершён",
  cancelled: "Обмен отменён",
};

const getTitle = (exchange: TExchange) => {
  const participant = exchange.participants[0];

  return participant
    ? `${participant.gives_item.title} → ${participant.receives_item.title}`
    : "Обмен без участников";
};

const PostComponent = ({ exchange }: TProps) => {
  const navigate = useNavigate();
  const exchangePath = `/exchanges/${exchange.id}`;

  const openExchange = () => navigate(exchangePath);
  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openExchange();
    }
  };

  return (
    <article
      className={styles.post}
      role="link"
      tabIndex={0}
      onClick={openExchange}
      onKeyDown={handleKeyDown}
    >
      <div className={styles.post__top}>
        <span
          className={`${styles.post__status} ${styles[`post__status_${exchange.status}`]}`}
        >
          {statusLabels[exchange.status]}
        </span>
        <span className={styles.post__participantsCount}>
          {exchange.participants.length} участников
        </span>
      </div>

      <h2 className={styles.post__title}>{getTitle(exchange)}</h2>

      <div className={styles.post__participants}>
        {exchange.participants.slice(0, 4).map(({ user }) => (
          <Link
            aria-label={`Профиль ${user.nickname}`}
            className={styles.post__participant}
            key={user.id}
            title={user.nickname}
            to={`/profile/${user.id}`}
            onClick={(event) => event.stopPropagation()}
            onKeyDown={(event) => event.stopPropagation()}
          >
            {user.photo_url ? (
              <img
                className={styles.post__participantImage}
                alt={user.nickname}
                src={user.photo_url}
              />
            ) : (
              user.nickname.charAt(0).toUpperCase()
            )}
          </Link>
        ))}
      </div>

      <span className={styles.post__open}>Открыть обмен</span>
    </article>
  );
};

export const Post = memo(PostComponent);
