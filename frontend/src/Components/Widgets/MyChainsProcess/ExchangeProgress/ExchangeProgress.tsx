import { memo } from "react";
import clsx from "clsx";

import styles from "./Styles.module.scss";
import type { TExchangeStatus } from "../../../../Api/exchanges/exchanges.types";
import { exchangeStatusPresentation } from "../../../../Features/Exchange/exchangeStatus";

type TProps = {
  status: TExchangeStatus;
  compact?: boolean;
};

const steps = [
  "Предложение",
  "Подтверждение",
  "Передача в ПВЗ",
  "Доставка",
  "Получение",
];
const ExchangeProgressComponent = ({ status, compact }: TProps) => {
  const current = exchangeStatusPresentation[status].progressStep;

  return (
    <div className={clsx(styles.progress, compact && styles.progress_compact)}>
      {steps.map((step, index) => {
        const completed = index < current;
        const active = status !== "completed" && index === current;

        return (
          <div
            className={clsx(
              styles.progress__step,
              completed && styles.progress__step_completed,
              active && styles.progress__step_active,
            )}
            key={step}
          >
            <span className={styles.progress__point}>
              {completed ? "✓" : ""}
            </span>
            <span className={styles.progress__label}>{step}</span>
          </div>
        );
      })}
    </div>
  );
};

export const ExchangeProgress = memo(ExchangeProgressComponent);
