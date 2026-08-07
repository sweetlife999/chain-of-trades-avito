import { memo, useEffect, type MouseEvent } from "react";

import styles from "./Styles.module.scss";
import { Button } from "../Button/Button";

type TProps = {
  title: string;
  description: string;
  confirmLabel: string;
  isPending?: boolean;
  error?: string;
  onClose: () => void;
  onConfirm: () => void;
};

const ConfirmationPopupComponent = ({
  title,
  description,
  confirmLabel,
  isPending = false,
  error,
  onClose,
  onConfirm,
}: TProps) => {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !isPending) {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isPending, onClose]);

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget && !isPending) {
      onClose();
    }
  };

  return (
    <div className={styles.confirmation__overlay} onClick={handleOverlayClick}>
      <section
        aria-labelledby="confirmation-title"
        aria-modal="true"
        className={styles.confirmation}
        role="dialog"
      >
        <div className={styles.confirmation__content}>
          <h2 id="confirmation-title">{title}</h2>
          <p>{description}</p>
          {error && <p className={styles.confirmation__error}>{error}</p>}
        </div>

        <div className={styles.confirmation__actions}>
          <Button
            color="transparent"
            disabled={isPending}
            type="button"
            onClick={onClose}
          >
            Назад
          </Button>
          <button
            className={styles.confirmation__danger}
            disabled={isPending}
            type="button"
            onClick={onConfirm}
          >
            {isPending ? "Отменяем..." : confirmLabel}
          </button>
        </div>
      </section>
    </div>
  );
};

export const ConfirmationPopup = memo(ConfirmationPopupComponent);
