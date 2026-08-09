import { useState } from "react";
import { PackageCheck } from "lucide-react";

import styles from "./Styles.module.scss";
import { Input } from "../../../UI/Input/Input";
import { MarkDeliveredButton } from "../MarkDeliveredButton/MarkDeliveredButton";

const UUID_PATTERN =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export const AdminExchangeDelivery = () => {
  const [exchangeId, setExchangeId] = useState("");
  const normalizedId = exchangeId.trim();
  const idError =
    normalizedId && !UUID_PATTERN.test(normalizedId)
      ? "Введите корректный UUID обмена"
      : undefined;
  const canSubmit = Boolean(normalizedId) && !idError;

  return (
    <section
      aria-labelledby="admin-exchange-delivery-title"
      className={styles.delivery}
    >
      <div className={styles.delivery__heading}>
        <span className={styles.delivery__icon} aria-hidden="true">
          <PackageCheck size={22} />
        </span>
        <div className={styles.delivery__headingContent}>
          <h2 className={styles.delivery__title} id="admin-exchange-delivery-title">
            Завершение доставки
          </h2>
          <p className={styles.delivery__description}>
            Переводит обмен из доставки к получению после того, как все вещи
            находятся в ПВЗ.
          </p>
        </div>
      </div>

      <div className={styles.delivery__form}>
        <Input
          autoComplete="off"
          error={idError}
          label="ID обмена"
          name="exchange-id"
          placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
          value={exchangeId}
          onChange={(event) => setExchangeId(event.target.value)}
        />
        <div className={styles.delivery__action}>
          {canSubmit ? (
            <MarkDeliveredButton
              exchangeId={normalizedId}
              onDelivered={() => setExchangeId("")}
            />
          ) : (
            <button
              className={styles.delivery__disabledButton}
              disabled
              type="button"
            >
              Завершить доставку
            </button>
          )}
        </div>
      </div>

      <p className={styles.delivery__hint}>
        Сервер отклонит запрос, если обмен ещё не находится в статусе доставки
        или не все вещи переданы в ПВЗ.
      </p>
    </section>
  );
};
