import { memo, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import clsx from "clsx";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../../Api/exchanges/exchanges";
import type { TExchange } from "../../../../Api/exchanges/exchanges.types";
import {
  exchangeStatusPresentation,
  type TExchangeStatusTab,
} from "../../../../Features/Exchange/exchangeStatus";
import { useAuthSelector } from "../../../../Hooks/useAuthDispatch";
import { ExchangeProgress } from "../ExchangeProgress/ExchangeProgress";
import { Button } from "../../../UI/Button/Button";
import { useMascot } from "../../../../Hooks/useMascot";

const tabs: { value: TExchangeStatusTab; label: string }[] = [
  { value: "active", label: "Активные" },
  { value: "completed", label: "Завершённые" },
  { value: "cancelled", label: "Отменённые" },
];

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
  const { isAuth, user } = useAuthSelector();
  const [tab, setTab] = useState<TExchangeStatusTab>("active");
  const { anchor, mood, movement, reactTo, reset } = useMascot();

  const previousExchangeIdsRef = useRef<{
    userId: string | null;
    idsByTab: Map<TExchangeStatusTab, Set<string>>;
  }>({
    userId: null,
    idsByTab: new Map(),
  });

  const {
    data = [],
    isPending,
    isError,
  } = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
    refetchInterval: 10_000,
    refetchIntervalInBackground: false,
  });

  const exchanges = useMemo(
    () =>
      data.filter(
        (exchange) =>
          exchange.participants.some(
            (participant) => participant.user.id === user?.id,
          ) && exchangeStatusPresentation[exchange.status].tab === tab,
      ),
    [data, tab, user?.id],
  );

  const exchangeIds = exchanges.map(({ id }) => id).join(":");

  const emptyChainsReactionActive =
    anchor === "chains-list" && mood === "bored" && movement === "wander";

  useEffect(() => {
    if (isPending || isError || !user?.id) {
      return;
    }

    const tracker = previousExchangeIdsRef.current;
    const currentIds = new Set(exchanges.map(({ id }) => id));

    if (tracker.userId !== user.id) {
      tracker.userId = user.id;
      tracker.idsByTab = new Map([[tab, currentIds]]);
      if (exchanges.length === 0) {
        if (!emptyChainsReactionActive) {
          reactTo("EMPTY_CHAINS");
        }
      } else if (emptyChainsReactionActive) {
        reset();
      }
      return;
    }

    const previousIds = tracker.idsByTab.get(tab);
    tracker.idsByTab.set(tab, currentIds);

    if (!previousIds) {
      if (exchanges.length === 0) {
        if (!emptyChainsReactionActive) {
          reactTo("EMPTY_CHAINS");
        }
      } else if (emptyChainsReactionActive) {
        reset();
      }
      return;
    }

    if (exchanges.some(({ id }) => !previousIds.has(id))) {
      reactTo("CHAIN_AVAILABLE");
      return;
    }

    if (exchanges.length === 0) {
      if (!emptyChainsReactionActive) {
        reactTo("EMPTY_CHAINS");
      }
    } else if (emptyChainsReactionActive) {
      reset();
    }
  }, [
    emptyChainsReactionActive,
    exchangeIds,
    exchanges,
    isError,
    isPending,
    reactTo,
    reset,
    tab,
    user?.id,
  ]);

  useEffect(() => {
    if (isError) {
      reactTo("ERROR");
    }
  }, [isError, reactTo]);

  return (
    <section className={styles.chains}>
      <div className={styles.chains__header}>
        <div className={styles.chains__heading}>
          <h1 className={styles.chains__title}>Мои цепочки</h1>
          <p className={styles.chains__description}>
            Следите за подтверждением, передачей в ПВЗ и доставкой вещей.
          </p>
        </div>
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
        <div className={styles.chains__list} data-mascot-anchor="chains-list">
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
                    {exchangeStatusPresentation[exchange.status].listLabel}
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
