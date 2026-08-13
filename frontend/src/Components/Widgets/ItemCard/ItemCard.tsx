import { memo } from "react";
import { Link } from "react-router-dom";

import styles from "./Styles.module.scss";
import type { TItem } from "../../../Api/items/items.types";
import { useCategoryName } from "../../../Hooks/useCategoryName";
import { ItemStatusBadge } from "../ItemStatusBadge/ItemStatusBadge";

type TProps = {
  item: TItem;
};

const ItemCardComponent = ({ item }: TProps) => {
  const categoryName = useCategoryName();

  return (
    <Link className={styles.card} to={`/items/${item.id}`}>
      <div className={styles.card__imageBox}>
        {item.photo_urls[0] ? (
          <img className={styles.card__image} src={item.photo_urls[0]} alt={item.title} />
        ) : (
          <span className={styles.card__imagePlaceholder}>Нет фото</span>
        )}
        <ItemStatusBadge className={styles.card__badge} item={item} />
      </div>

      <div className={styles.card__content}>
        <h2 className={styles.card__title}>{item.title}</h2>
        <span className={styles.card__category}>{categoryName(item.category)}</span>
        <p className={styles.card__description}>{item.description}</p>
      </div>
    </Link>
  );
};

export const ItemCard = memo(ItemCardComponent);
