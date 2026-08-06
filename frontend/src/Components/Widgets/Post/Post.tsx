import { memo } from "react";
import { Link } from "react-router-dom";

import styles from "./Styles.module.scss";
import type {
  TItem,
  TItemStatus,
} from "../../../Api/items/items.types";

type TPostProps = {
  post: TItem;
};

const statusLabels: Record<TItemStatus, string> = {
  available: "Доступно",
  reserved: "Зарезервировано",
  traded: "Обменяно",
  withdrawn: "Снято",
};

const PostComponent = ({ post }: TPostProps) => (
  <Link className={styles.post} to={`/items/${post.id}`}>
    <div className={styles.post__imageBox}>
      {post.photo_urls[0] ? (
        <img
          className={styles.post__image}
          src={post.photo_urls[0]}
          alt={post.title}
        />
      ) : (
        <span className={styles.post__placeholder}>Нет фото</span>
      )}

      <span
        className={`${styles.post__status} ${styles[`post__status_${post.status}`]}`}
      >
        {statusLabels[post.status]}
      </span>
    </div>

    <div className={styles.post__content}>
      <div className={styles.post__heading}>
        <h3 className={styles.post__title}>{post.title}</h3>
        <span className={styles.post__category}>{post.category}</span>
      </div>

      <p className={styles.post__description}>{post.description}</p>

      <div className={styles.post__wants}>
        {post.wants.map((want) => (
          <span key={want}>{want}</span>
        ))}
      </div>
    </div>
  </Link>
);

export const Post = memo(PostComponent);