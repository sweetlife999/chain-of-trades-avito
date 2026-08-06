import { memo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import { getItem } from "../../../Api/items/items";
import type { TItemStatus } from "../../../Api/items/items.types";

const labels: Record<TItemStatus, string> = {
  available: "Доступно для обмена",
  reserved: "Участвует в цепочке",
  traded: "Обменяно",
  withdrawn: "Снято с публикации",
};

const ItemDetailsComponent = () => {
  const { id = "" } = useParams();
  const { data: item, isPending, isError } = useQuery({
    queryKey: ["items", id],
    queryFn: () => getItem(id),
    enabled: Boolean(id),
  });

  if (isPending) return <p>Загрузка...</p>;
  if (isError || !item) return <p>Не удалось загрузить вещь</p>;

  return (
    <article className={styles.item}>
      <Link to="/myItems">← Мои вещи</Link>
      <div className={styles.item__grid}>
        <div className={styles.item__photos}>
          {item.photo_urls.map((url) => <img key={url} src={url} alt={item.title} />)}
        </div>
        <div className={styles.item__info}>
          <span className={styles[`status_${item.status}`]}>{labels[item.status]}</span>
          <h1>{item.title}</h1>
          <small>{item.category}</small>
          <p>{item.description}</p>
          <h2>Хочу получить</h2>
          <div>{item.wants.map((want) => <span key={want}>{want}</span>)}</div>
        </div>
      </div>
    </article>
  );
};

export const ItemDetails = memo(ItemDetailsComponent);
