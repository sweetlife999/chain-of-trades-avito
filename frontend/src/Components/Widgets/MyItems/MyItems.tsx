import { memo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import clsx from "clsx";

import styles from "./Styles.module.scss";
import { getItems } from "../../../Api/items/items";
import type { TItemStatus } from "../../../Api/items/items.types";
import { ItemCard } from "../ItemCard/ItemCard";

type TFilter = "all" | TItemStatus;

const filters: { value: TFilter; label: string }[] = [
  { value: "all", label: "Все" },
  { value: "available", label: "Доступные" },
  { value: "reserved", label: "В цепочке" },
  { value: "traded", label: "Обменянные" },
  { value: "withdrawn", label: "Снятые" },
];

const MyItemsComponent = () => {
  const [filter, setFilter] = useState<TFilter>("all");
  const { data = [], isPending, isError } = useQuery({
    queryKey: ["items"],
    queryFn: getItems,
  });
  const items = filter === "all" ? data : data.filter(({ status }) => status === filter);

  return (
    <section className={styles.items}>
      <header className={styles.items__header}>
        <h1>Мои вещи</h1>
        <Link to="/exchanges/create">Добавить вещь</Link>
      </header>

      <div className={styles.items__filters}>
        {filters.map(({ value, label }) => (
          <button
            className={clsx(filter === value && styles.active)}
            key={value}
            type="button"
            onClick={() => setFilter(value)}
          >
            {label}
          </button>
        ))}
      </div>

      {isPending && <p>Загрузка...</p>}
      {isError && <p>Не удалось загрузить вещи</p>}
      {!isPending && !isError && (
        <div className={styles.items__grid}>
          {items.length ? items.map((item) => <ItemCard key={item.id} item={item} />) : <p>Вещей пока нет</p>}
        </div>
      )}
    </section>
  );
};

export const MyItems = memo(MyItemsComponent);
