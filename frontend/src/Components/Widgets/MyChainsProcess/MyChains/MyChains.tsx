import { memo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../../Api/exchanges/exchanges";
import type { TExchange } from "../../../../Api/exchanges/exchanges.types";
import { getItems } from "../../../../Api/items/items";
import type { TItemStatus } from "../../../../Api/items/items.types";
import { Post } from "../../../Widgets/Post/Post";

type TTab = "items" | "active" | "completed" | "cancelled";
type TItemFilter = "all" | TItemStatus;
type TExchangeStatus = "proposed" | "confirmed" | "completed" | "cancelled";

const tabs: { value: TTab; label: string }[] = [
  { value: "items", label: "Мои объявления" },
  { value: "active", label: "Активные обмены" },
  { value: "completed", label: "Завершённые" },
  { value: "cancelled", label: "Отменённые" },
];

const itemFilters: { value: TItemFilter; label: string }[] = [
  { value: "all", label: "Все" },
  { value: "available", label: "Доступные" },
  { value: "reserved", label: "Зарезервированные" },
  { value: "traded", label: "Обменянные" },
  { value: "withdrawn", label: "Снятые" },
];

const exchangeStatuses: Record<
  TExchangeStatus,
  { label: string; tab: Exclude<TTab, "items"> }
> = {
  proposed: { label: "Ожидает подтверждения", tab: "active" },
  confirmed: { label: "Обмен подтверждён", tab: "active" },
  completed: { label: "Обмен завершён", tab: "completed" },
  cancelled: { label: "Обмен отменён", tab: "cancelled" },
};

const getExchangeTitle = (exchange: TExchange) => {
  const participant = exchange.participants[0];

  return participant
    ? `${participant.gives_item.title} → ${participant.receives_item.title}`
    : "Обмен без участников";
};

const MyChainsComponent = () => {
  const [activeTab, setActiveTab] = useState<TTab>("items");
  const [itemFilter, setItemFilter] = useState<TItemFilter>("all");

  const itemsQuery = useQuery({ queryKey: ["items"], queryFn: getItems });
  const exchangesQuery = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
  });

  const items = itemsQuery.data ?? [];
  const exchanges = exchangesQuery.data ?? [];
  const filteredItems =
    itemFilter === "all"
      ? items
      : items.filter((item) => item.status === itemFilter);
  const filteredExchanges = exchanges.filter((exchange) => {
    const config = exchangeStatuses[exchange.status as TExchangeStatus];
    return config?.tab === activeTab;
  });

  return (
    <main className={styles.chains}>
      <div className={styles.chains__header}>
        <h1>Мои объявления и обмены</h1>
        <Link className={styles.chains__create} to="/exchanges/create">
          Создать объявление
        </Link>
      </div>

      <div className={styles.chains__tabs}>
        {tabs.map((tab) => (
          <button
            key={tab.value}
            className={activeTab === tab.value ? styles.active : ""}
            type="button"
            onClick={() => setActiveTab(tab.value)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === "items" ? (
        <>
          <div className={styles.chains__filters}>
            {itemFilters.map((filter) => (
              <button
                key={filter.value}
                className={itemFilter === filter.value ? styles.active : ""}
                type="button"
                onClick={() => setItemFilter(filter.value)}
              >
                {filter.label}
              </button>
            ))}
          </div>

          {itemsQuery.isPending && <p>Загрузка объявлений...</p>}
          {itemsQuery.isError && <p>Не удалось загрузить объявления</p>}
          {!itemsQuery.isPending && !itemsQuery.isError && (
            <div className={styles.chains__items}>
              {filteredItems.map((item) => (
                <Post key={item.id} post={item} />
              ))}
            </div>
          )}
        </>
      ) : (
        <>
          {exchangesQuery.isPending && <p>Загрузка обменов...</p>}
          {exchangesQuery.isError && <p>Не удалось загрузить обмены</p>}
          {!exchangesQuery.isPending && !exchangesQuery.isError && (
            <div className={styles.chains__exchanges}>
              {filteredExchanges.map((exchange) => {
                const status = exchange.status as TExchangeStatus;

                return (
                  <Link
                    key={exchange.id}
                    className={styles.chains__exchange}
                    to={`/exchanges/${exchange.id}`}
                  >
                    <h2>{getExchangeTitle(exchange)}</h2>
                    <span className={styles[`status_${status}`]}>
                      {exchangeStatuses[status]?.label ?? exchange.status}
                    </span>
                  </Link>
                );
              })}
            </div>
          )}
        </>
      )}
    </main>
  );
};

export const MyChains = memo(MyChainsComponent);