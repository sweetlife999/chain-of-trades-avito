import { memo } from "react";
import { Link } from "react-router-dom";

import styles from "./Styles.module.scss";
import type { TItem, TItemStatus } from "../../../Api/items/items.types";

type TProps = {
  item: TItem;
};

const labels: Record<TItemStatus, string> = {
  available: "Доступно",
  reserved: "В цепочке",
  traded: "Обменяно",
  withdrawn: "Снято",
};

const ItemCardComponent = ({ item }: TProps) => (
  <Link className={styles.card} to={`/items/${item.id}`}>
    <div className={styles.card__imageBox}>
      {item.photo_urls[0] ? (
        <img className={styles.card__image} src={item.photo_urls[0]} alt={item.title} />
      ) : (
        <span className={styles.card__imagePlaceholder}>Нет фото</span>
      )}
      <span className={`${styles.card__status} ${styles[`card__status_${item.status}`]}`}>
        {labels[item.status]}
      </span>
    </div>

    <div className={styles.card__content}>
      <h2 className={styles.card__title}>{item.title}</h2>
      <span className={styles.card__category}>{item.category}</span>
      {item.pickup_point && (
        <span className={styles.card__pickup}>
          В ПВЗ: {item.pickup_point.name}
        </span>
      )}
      <p className={styles.card__description}>{item.description}</p>
    </div>
  </Link>
);

export const ItemCard = memo(ItemCardComponent);
