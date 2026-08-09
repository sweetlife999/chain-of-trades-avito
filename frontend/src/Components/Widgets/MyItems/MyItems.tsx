import { memo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import clsx from "clsx";

import styles from "./Styles.module.scss";
import { getItems } from "../../../Api/items/items";
import type { TItemStatus } from "../../../Api/items/items.types";
import { ItemCard } from "../ItemCard/ItemCard";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { AuthRequiredState } from "../../UI/AuthRequiredState/AuthRequiredState";

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
  const { isAuth } = useAuthSelector();
  const { data = [], isPending, isError } = useQuery({
    queryKey: ["items"],
    queryFn: getItems,
    enabled: isAuth,
  });
  const items = filter === "all" ? data : data.filter(({ status }) => status === filter);

  return (
    <section className={styles.items}>
      <div className={styles.items__header}>
        <div className={styles.items__heading}>
          <h1 className={styles.items__title}>Мои вещи</h1>
          {isAuth && (
            <p className={styles.items__description}>
              Управляйте объявлениями и отслеживайте состояние каждой вещи.
            </p>
          )}
        </div>
        {isAuth && (
          <Link className={styles.items__create} to="/exchanges/create">
            Добавить вещь
          </Link>
        )}
      </div>

      {!isAuth ? (
        <AuthRequiredState
          description="Создайте аккаунт или войдите, чтобы добавлять вещи, менять их статус и участвовать в цепочках обмена."
          returnTo="/myItems"
          title="Войдите, чтобы управлять своими вещами"
        />
      ) : (
        <>
          <div className={styles.items__filters}>
            {filters.map(({ value, label }) => (
              <button
                className={clsx(
                  styles.items__filter,
                  filter === value && styles.items__filter_active,
                )}
                key={value}
                type="button"
                onClick={() => setFilter(value)}
              >
                {label}
              </button>
            ))}
          </div>

          {isPending && (
            <p className={styles.items__state}>Загружаем ваши вещи...</p>
          )}
          {isError && (
            <p className={styles.items__state_error}>
              Не удалось загрузить вещи. Попробуйте обновить страницу.
            </p>
          )}
          {!isPending && !isError && (
            <div className={styles.items__grid}>
              {items.length ? (
                items.map((item) => <ItemCard key={item.id} item={item} />)
              ) : (
                <p className={styles.items__state}>
                  В этой категории вещей пока нет.
                </p>
              )}
            </div>
          )}
        </>
      )}
    </section>
  );
};

export const MyItems = memo(MyItemsComponent);
