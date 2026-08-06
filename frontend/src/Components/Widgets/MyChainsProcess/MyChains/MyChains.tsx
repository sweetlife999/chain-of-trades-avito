import { memo, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import styles from "./Styles.module.scss";

import { Button } from "../../../UI/Button/Button";
import { getExchanges } from "../../../../Api/exchanges/exchanges";
import type { TExchange } from "../../../../Api/exchanges/exchanges.types";

type TTab = "active" | "completed" | "cancelled";
type TExchangeStatus =
  | "proposed"
  | "confirmed"
  | "completed"
  | "cancelled";

type TStatusConfig = {
  label: string;
  tab: TTab;
  progressStep: number | null;
};

const tabs: { value: TTab; label: string }[] = [
  {
    value: "active",
    label: "Активные",
  },
  {
    value: "completed",
    label: "Завершённые",
  },
  {
    value: "cancelled",
    label: "Отменённые",
  },
];

const statusConfig: Record<TExchangeStatus, TStatusConfig> = {
  proposed: {
    label: "Ожидает подтверждения",
    tab: "active",
    progressStep: 0,
  },
  confirmed: {
    label: "Обмен подтверждён",
    tab: "active",
    progressStep: 1,
  },
  completed: {
    label: "Обмен завершён",
    tab: "completed",
    progressStep: 2,
  },
  cancelled: {
    label: "Обмен отменён",
    tab: "cancelled",
    progressStep: null,
  },
};

const progressSteps = ["Обмен найден", "Подтверждён", "Завершён"];

const normalizeExchangeStatus = (
  status: string,
): TExchangeStatus | null => {
  const normalizedStatus = status.toLowerCase();

  if (normalizedStatus in statusConfig) {
    return normalizedStatus as TExchangeStatus;
  }

  return null;
};

const getExchangeTab = (status: string): TTab => {
  const normalizedStatus = normalizeExchangeStatus(status);

  return normalizedStatus
    ? statusConfig[normalizedStatus].tab
    : "active";
};

const getStatusLabel = (status: string) => {
  const normalizedStatus = normalizeExchangeStatus(status);

  return normalizedStatus
    ? statusConfig[normalizedStatus].label
    : "Неизвестный статус";
};

const getStatusClassName = (status: string) => {
  const normalizedStatus = normalizeExchangeStatus(status);

  if (!normalizedStatus) {
    return styles.chains__status;
  }

  return `${styles.chains__status} ${
    styles[`chains__status_${normalizedStatus}`]
  }`;
};

const getExchangeTitle = (exchange: TExchange) => {
  const participant = exchange.participants[0];

  if (!participant) {
    return "Обмен без участников";
  }

  return `${participant.gives_item.title} → ${participant.receives_item.title}`;
};

const formatDate = (date: string) => {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  }).format(new Date(date));
};

const MyChainsComponent = () => {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<TTab>("active");

  const {
    data: exchanges = [],
    isPending,
    isError,
  } = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
  });

  const filteredExchanges = useMemo(() => {
    return exchanges.filter(
      (exchange) => getExchangeTab(exchange.status) === activeTab,
    );
  }, [exchanges, activeTab]);

  return (
    <main className={styles.chains}>
      <div className={styles.chains__header}>
        <h1 className={styles.chains__title}>Мои цепочки</h1>

        <Button
          color="light"
          type="button"
          onClick={() => navigate("/exchanges/create")}
        >
          Создать цепочку
        </Button>
      </div>

      <div className={styles.chains__tabs} role="tablist">
        {tabs.map((tab) => (
          <button
            key={tab.value}
            className={`${styles.chains__tab} ${
              activeTab === tab.value
                ? styles.chains__tab_active
                : ""
            }`}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.value}
            onClick={() => setActiveTab(tab.value)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {isPending && (
        <p className={styles.chains__message}>Загрузка цепочек...</p>
      )}

      {isError && (
        <p className={styles.chains__message}>
          Не удалось загрузить цепочки
        </p>
      )}

      {!isPending && !isError && (
        <div className={styles.chains__list}>
          {filteredExchanges.length > 0 ? (
            filteredExchanges.map((exchange) => {
              const normalizedStatus = normalizeExchangeStatus(
                exchange.status,
              );
              const currentStep = normalizedStatus
                ? statusConfig[normalizedStatus].progressStep
                : 0;
              const isCancelled = normalizedStatus === "cancelled";

              return (
                <article
                  key={exchange.id}
                  className={styles.chains__card}
                >
                  <div className={styles.chains__cardInformation}>
                    <h2 className={styles.chains__cardTitle}>
                      {getExchangeTitle(exchange)}
                    </h2>

                    <span className={getStatusClassName(exchange.status)}>
                      {getStatusLabel(exchange.status)}
                    </span>

                    <time
                      className={styles.chains__date}
                      dateTime={exchange.created_at}
                    >
                      {formatDate(exchange.created_at)}
                    </time>
                  </div>

                  <div className={styles.chains__cardContent}>
                    {isCancelled ? (
                      <div className={styles.chains__cancelledNotice}>
                        Эта цепочка больше не участвует в обмене.
                      </div>
                    ) : (
                      <div className={styles.chains__progress}>
                        <div className={styles.chains__progressLine}>
                          <span
                            className={styles.chains__progressCompleted}
                            style={{
                              width: `${
                                ((currentStep ?? 0) /
                                  (progressSteps.length - 1)) *
                                100
                              }%`,
                            }}
                          />
                        </div>

                        <div className={styles.chains__progressSteps}>
                          {progressSteps.map((step, index) => (
                            <div
                              key={step}
                              className={styles.chains__progressStep}
                            >
                              <span
                                className={`${styles.chains__progressPoint} ${
                                  index <= (currentStep ?? 0)
                                    ? styles.chains__progressPoint_active
                                    : ""
                                }`}
                              />

                              <span
                                className={styles.chains__progressLabel}
                              >
                                {step}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    <Button
                      color="light"
                      type="button"
                      onClick={() =>
                        navigate(`/exchanges/${exchange.id}`)
                      }
                    >
                      Открыть
                    </Button>
                  </div>
                </article>
              );
            })
          ) : (
            <div className={styles.chains__empty}>
              <h2 className={styles.chains__emptyTitle}>
                Цепочек пока нет
              </h2>

              <p className={styles.chains__emptyDescription}>
                В этой категории пока нет обменов.
              </p>
            </div>
          )}
        </div>
      )}
    </main>
  );
};

export const MyChains = memo(MyChainsComponent);