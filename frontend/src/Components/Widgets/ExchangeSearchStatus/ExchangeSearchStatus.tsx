import { Radar } from "lucide-react";

import styles from "./Styles.module.scss";

type TExchangeSearchStatusProps = {
  exchangesCount: number;
  isLoading: boolean;
  hasOwnItems: boolean;
};

export const ExchangeSearchStatus = ({
  exchangesCount,
  isLoading,
  hasOwnItems,
}: TExchangeSearchStatusProps) => (
  <aside className={styles.searchStatus} aria-live="polite">
    <span className={styles.searchStatus__icon} aria-hidden="true">
      <Radar className={styles.searchStatus__iconGraphic} />
    </span>

    <div className={styles.searchStatus__content}>
      <strong className={styles.searchStatus__title}>
        {isLoading ? "Обновляем предложения" : "Поиск обменов всегда активен"}
      </strong>

      <p className={styles.searchStatus__description}>
        {exchangesCount > 0
          ? `Сейчас найдено предложений: ${exchangesCount}. Мы продолжим искать новые варианты.`
          : "Можно заниматься своими делами — подходящая цепочка появится здесь автоматически."}
      </p>
    </div>

    {hasOwnItems && (
      <span className={styles.searchStatus__activity}>
        <span className={styles.searchStatus__pulse} aria-hidden="true" />
        Поиск идёт
      </span>
    )}
  </aside>
);
