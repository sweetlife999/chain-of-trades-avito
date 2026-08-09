import { useEffect, useState, type MouseEvent } from "react";
import { ShieldCheck, ShieldX, X } from "lucide-react";

import { Button } from "../../../UI/Button/Button";
import styles from "./Styles.module.scss";

export type TAdminReportDecisionType = "reject" | "resolve";

type TProps = {
  error?: string;
  isPending: boolean;
  onClose: () => void;
  onConfirm: (comment: string) => void;
  type: TAdminReportDecisionType;
};

export const AdminReportDecision = ({
  error,
  isPending,
  onClose,
  onConfirm,
  type,
}: TProps) => {
  const [comment, setComment] = useState("");
  const resolve = type === "resolve";

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
    <div className={styles.decision__overlay} onMouseDown={handleOverlayClick}>
      <section
        aria-labelledby="report-decision-title"
        aria-modal="true"
        className={styles.decision}
        role="dialog"
      >
        <header className={styles.decision__header}>
          <span
            className={[
              styles.decision__icon,
              resolve
                ? styles.decision__icon_resolve
                : styles.decision__icon_reject,
            ].join(" ")}
            aria-hidden="true"
          >
            {resolve ? <ShieldCheck /> : <ShieldX />}
          </span>
          <div className={styles.decision__heading}>
            <h2 className={styles.decision__title} id="report-decision-title">
              {resolve ? "Подтвердить жалобу?" : "Отклонить жалобу?"}
            </h2>
            <p className={styles.decision__description}>
              Решение закроет жалобу и будет сохранено в истории модерации.
            </p>
          </div>
          <button
            aria-label="Закрыть"
            className={styles.decision__close}
            disabled={isPending}
            onClick={onClose}
            type="button"
          >
            <X aria-hidden="true" />
          </button>
        </header>

        <label className={styles.decision__field}>
          <span className={styles.decision__label}>
            Комментарий администратора
          </span>
          <textarea
            className={styles.decision__textarea}
            disabled={isPending}
            onChange={(event) => setComment(event.target.value)}
            placeholder={
              resolve
                ? "Например: нарушение подтверждено"
                : "Например: нарушение не обнаружено"
            }
            rows={4}
            value={comment}
          />
        </label>

        {error && (
          <p className={styles.decision__error} role="alert">
            {error}
          </p>
        )}

        <div className={styles.decision__actions}>
          <Button
            className={styles.decision__action}
            color="transparent"
            disabled={isPending}
            onClick={onClose}
          >
            Назад
          </Button>
          <Button
            className={styles.decision__action}
            color={resolve ? "green" : "danger"}
            disabled={isPending}
            onClick={() => onConfirm(comment.trim())}
          >
            {isPending
              ? "Сохраняем решение..."
              : resolve
                ? "Подтвердить"
                : "Отклонить"}
          </Button>
        </div>
      </section>
    </div>
  );
};
