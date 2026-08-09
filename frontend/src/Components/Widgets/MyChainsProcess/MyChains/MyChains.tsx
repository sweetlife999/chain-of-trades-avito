import { memo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import clsx from "clsx";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../../Api/exchanges/exchanges";
import type {
  TExchange,
  TExchangeStatus,
} from "../../../../Api/exchanges/exchanges.types";
import { useAuthSelector } from "../../../../Hooks/useAuthDispatch";
import { ExchangeProgress } from "../ExchangeProgress/ExchangeProgress";
import { Button } from "../../../UI/Button/Button";

type TTab = "active" | "completed" | "cancelled";

const tabs: { value: TTab; label: string }[] = [
  { value: "active", label: "Активные" },
  { value: "completed", label: "Завершённые" },
  { value: "cancelled", label: "Отменённые" },
];

const statusData: Record<TExchangeStatus, { label: string; tab: TTab }> = {
  proposed: { label: "Ждём вашего подтверждения", tab: "active" },
  confirmed: { label: "Передача вещи в ПВЗ", tab: "active" },
  delivering: { label: "Вещи доставляются между ПВЗ", tab: "active" },
  delivered: { label: "Вещь ожидает вас в ПВЗ", tab: "active" },
  completed: { label: "Обмен завершён", tab: "completed" },
  cancelled: { label: "Цепочка распалась", tab: "cancelled" },
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU").format(new Date(value));

const getTitle = (exchange: TExchange, userId?: string) => {
  const participant =
    exchange.participants.find(({ user }) => user.id === userId) ??
    exchange.participants[0];

  return participant
    ? `${participant.gives_item.title} → ${participant.receives_item.title}`
    : "Обмен без участников";
};

const MyChainsComponent = () => {
  const navigate = useNavigate();
  const { isAuth } = useAuthSelector();
  const [tab, setTab] = useState<TTab>("active");
  const { user } = useAuthSelector();
  const {
    data = [],
    isPending,
    isError,
  } = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
  });

  const exchanges = data.filter(
    (exchange) =>
      exchange.participants.some(
        (participant) => participant.user.id === user?.id,
      ) && statusData[exchange.status].tab === tab,
  );

  return (
    <section className={styles.chains}>
      <div className={styles.chains__header}>
        <div className={styles.chains__heading}>
          <h1 className={styles.chains__title}>Мои цепочки</h1>
          <p className={styles.chains__description}>
            Следите за подтверждением, передачей в ПВЗ и доставкой вещей.
          </p>
        </div>
        <Button
          className={styles.chains__create}
          onClick={() =>
            isAuth
              ? navigate("/exchanges/create")
              : navigate("/login", {
                  state: { from: "/exchanges/create" },
                })
          }
        >
          Создать цепочку
        </Button>
      </div>

      <div className={styles.chains__filters}>
            {tabs.map(({ value, label }) => (
              <button
                className={clsx(
                  styles.chains__filter,
                  tab === value && styles.chains__filter_active,
                )}
                key={value}
                type="button"
                onClick={() => setTab(value)}
              >
                {label}
              </button>
            ))}
          </div>

      {isPending && <p className={styles.chains__message}>Загрузка...</p>}
      {isError && (
        <p className={styles.chains__message_error}>
          Не удалось загрузить цепочки
        </p>
      )}

      {!isPending && !isError && (
        <div className={styles.chains__list}>
          {exchanges.length ? (
            exchanges.map((exchange) => (
              <Link
                className={styles.chains__card}
                key={exchange.id}
                to={`/exchanges/${exchange.id}`}
              >
                <div className={styles.chains__info}>
                  <h2 className={styles.chains__cardTitle}>
                    {getTitle(exchange, user?.id)}
                  </h2>
                  <span
                    className={`${styles.chains__status} ${
                      styles[`chains__status_${exchange.status}`]
                    }`}
                  >
                    {statusData[exchange.status].label}
                  </span>
                  <time
                    className={styles.chains__date}
                    dateTime={exchange.created_at}
                  >
                    {formatDate(exchange.created_at)}
                  </time>
                </div>

                <div className={styles.chains__progress}>
                  <ExchangeProgress compact status={exchange.status} />
                  <span className={styles.chains__open}>Открыть</span>
                </div>
              </Link>
            ))
          ) : (
            <p className={styles.chains__message}>Цепочек пока нет</p>
          )}
        </div>
      )}
    </section>
  );
};

export const MyChains = memo(MyChainsComponent);
