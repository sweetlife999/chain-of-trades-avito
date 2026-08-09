import { useEffect, useState, type FormEvent, type MouseEvent } from "react";
import { createPortal } from "react-dom";
import { useMutation } from "@tanstack/react-query";
import { CircleCheck } from "lucide-react";

import styles from "./Styles.module.scss";
import {
  createReport,
  getReportErrorMessage,
} from "../../../../Api/reports/reports";
import { Button } from "../../../UI/Button/Button";
import { Input } from "../../../UI/Input/Input";

type TReportMessagePopupProps = {
  messageId: string;
  onClose: () => void;
  onSubmitted: () => void;
};

export const ReportMessagePopup = ({
  messageId,
  onClose,
  onSubmitted,
}: TReportMessagePopupProps) => {
  const [reason, setReason] = useState("abuse");
  const [comment, setComment] = useState("");

  const reportMutation = useMutation({
    mutationFn: () =>
      createReport({
        comment: comment.trim() || undefined,
        message_id: messageId,
        reason: reason.trim(),
      }),
    onSuccess: onSubmitted,
  });

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !reportMutation.isPending) {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose, reportMutation.isPending]);

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget && !reportMutation.isPending) {
      onClose();
    }
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!reason.trim() || reportMutation.isPending) {
      return;
    }

    reportMutation.mutate();
  };

  return createPortal(
    <div className={styles.report__overlay} onClick={handleOverlayClick}>
      <section
        aria-labelledby="report-message-title"
        aria-modal="true"
        className={styles.report}
        role="dialog"
      >
        {reportMutation.isSuccess ? (
          <div className={styles.report__success}>
            <CircleCheck
              className={styles.report__successIcon}
              aria-hidden="true"
              size={42}
            />
            <h2
              className={styles.report__successTitle}
              id="report-message-title"
            >
              Жалоба отправлена
            </h2>
            <p className={styles.report__successDescription}>
              Она добавлена в очередь модерации. На подбор обменов жалоба не
              влияет.
            </p>
            <Button
              centered
              className={styles.report__successAction}
              color="light"
              type="button"
              onClick={onClose}
            >
              Понятно
            </Button>
          </div>
        ) : (
          <form className={styles.report__form} onSubmit={handleSubmit}>
            <header className={styles.report__header}>
              <h2 className={styles.report__title} id="report-message-title">
                Пожаловаться на сообщение
              </h2>
              <p className={styles.report__description}>
                Укажите причину и при необходимости добавьте пояснение для
                модератора.
              </p>
            </header>

            <Input
              required
              disabled={reportMutation.isPending}
              label="Причина жалобы"
              placeholder="Например: abuse"
              value={reason}
              onChange={(event) => {
                setReason(event.target.value);
                reportMutation.reset();
              }}
            />

            <Input
              textarea
              disabled={reportMutation.isPending}
              label="Комментарий"
              placeholder="Опишите, что не так с сообщением"
              rows={4}
              value={comment}
              onChange={(event) => {
                setComment(event.target.value);
                reportMutation.reset();
              }}
            />

            {reportMutation.isError && (
              <p className={styles.report__error} role="alert">
                {getReportErrorMessage(reportMutation.error)}
              </p>
            )}

            <div className={styles.report__actions}>
              <Button
                color="transparent"
                disabled={reportMutation.isPending}
                type="button"
                onClick={onClose}
              >
                Назад
              </Button>
              <Button
                color="danger"
                disabled={!reason.trim() || reportMutation.isPending}
                type="submit"
              >
                {reportMutation.isPending ? "Отправляем..." : "Отправить"}
              </Button>
            </div>
          </form>
        )}
      </section>
    </div>,
    document.body,
  );
};
