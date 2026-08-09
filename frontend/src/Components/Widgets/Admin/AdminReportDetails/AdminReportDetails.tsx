import { useEffect, useState, type MouseEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  AlertTriangle,
  CheckCircle2,
  MessageSquareText,
  ShieldCheck,
  UserRound,
  X,
  XCircle,
} from "lucide-react";

import {
  assignAdminReport,
  getAdminErrorMessage,
  getAdminReport,
  rejectAdminReport,
  resolveAdminReport,
} from "../../../../Api/admin/admin";
import type { TAdminReport } from "../../../../Api/admin/admin.types";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import {
  AdminReportDecision,
  type TAdminReportDecisionType,
} from "../AdminReportDecision/AdminReportDecision";
import { AdminReportMessages } from "../AdminReportMessages/AdminReportMessages";
import styles from "./Styles.module.scss";

type TProps = {
  currentAdminId: string;
  onClose: () => void;
  reportId: string;
};

const statusLabels = {
  open: "Открыта",
  rejected: "Отклонена",
  resolved: "Подтверждена",
} as const;

const reasonLabels = {
  abuse: "Оскорбления",
  other: "Другое",
  spam: "Спам",
} as const;

const formatDate = (value?: string | null) =>
  value
    ? new Intl.DateTimeFormat("ru-RU", {
        dateStyle: "medium",
        timeStyle: "short",
      }).format(new Date(value))
    : "—";

export const AdminReportDetails = ({
  currentAdminId,
  onClose,
  reportId,
}: TProps) => {
  const [decision, setDecision] =
    useState<TAdminReportDecisionType | null>(null);
  const [messagesOpen, setMessagesOpen] = useState(false);
  const queryClient = useQueryClient();
  const queryKey = ["admin", "reports", reportId] as const;

  const reportQuery = useQuery({
    queryKey,
    queryFn: () => getAdminReport(reportId),
    retry: false,
  });

  const refreshReportLists = () =>
    queryClient.invalidateQueries({ queryKey: ["admin", "reports", "list"] });

  const assignMutation = useMutation({
    mutationFn: () => assignAdminReport(reportId),
    onSuccess: async (report) => {
      queryClient.setQueryData<TAdminReport>(queryKey, report);
      await refreshReportLists();
    },
  });

  const decisionMutation = useMutation({
    mutationFn: ({
      comment,
      type,
    }: {
      comment: string;
      type: TAdminReportDecisionType;
    }) =>
      type === "resolve"
        ? resolveAdminReport(reportId, { comment })
        : rejectAdminReport(reportId, { comment }),
    onSuccess: async (report) => {
      queryClient.setQueryData<TAdminReport>(queryKey, report);
      setDecision(null);
      await refreshReportLists();
    },
  });

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        event.key === "Escape" &&
        !decision &&
        !assignMutation.isPending &&
        !decisionMutation.isPending
      ) {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [
    assignMutation.isPending,
    decision,
    decisionMutation.isPending,
    onClose,
  ]);

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (
      event.target === event.currentTarget &&
      !assignMutation.isPending &&
      !decisionMutation.isPending
    ) {
      onClose();
    }
  };

  const report = reportQuery.data;
  const assignedToCurrentAdmin = report?.assignee?.id === currentAdminId;
  const assignedToAnotherAdmin = Boolean(
    report?.assignee && !assignedToCurrentAdmin,
  );

  return (
    <div className={styles.reportDetails__overlay} onMouseDown={handleOverlayClick}>
      <section
        aria-labelledby="admin-report-title"
        aria-modal="true"
        className={styles.reportDetails}
        role="dialog"
      >
        <header className={styles.reportDetails__header}>
          <span className={styles.reportDetails__icon} aria-hidden="true">
            <AlertTriangle className={styles.reportDetails__iconImage} />
          </span>
          <div className={styles.reportDetails__heading}>
            <span className={styles.reportDetails__eyebrow}>
              Карточка жалобы
            </span>
            <h2 className={styles.reportDetails__title} id="admin-report-title">
              {report ? reasonLabels[report.reason] : "Загрузка жалобы"}
            </h2>
          </div>
          <button
            aria-label="Закрыть карточку жалобы"
            className={styles.reportDetails__close}
            disabled={assignMutation.isPending || decisionMutation.isPending}
            onClick={onClose}
            type="button"
          >
            <X aria-hidden="true" className={styles.reportDetails__closeIcon} />
          </button>
        </header>

        {reportQuery.isPending && (
          <div className={styles.reportDetails__state}>
            <Loader text="Загружаем жалобу..." />
          </div>
        )}

        {reportQuery.isError && (
          <div className={styles.reportDetails__state} role="alert">
            <strong className={styles.reportDetails__stateTitle}>
              Не удалось загрузить жалобу
            </strong>
            <Button
              color="light"
              size="s"
              onClick={() => void reportQuery.refetch()}
            >
              Повторить
            </Button>
          </div>
        )}

        {report && (
          <div className={styles.reportDetails__body}>
            <div className={styles.reportDetails__summary}>
              <span
                className={[
                  styles.reportDetails__status,
                  styles["reportDetails__status_" + report.status],
                ].join(" ")}
              >
                {statusLabels[report.status]}
              </span>
              <span className={styles.reportDetails__date}>
                Создана {formatDate(report.created_at)}
              </span>
              <span className={styles.reportDetails__exchange}>
                Обмен: {report.exchange.id} · {report.exchange.status}
              </span>
            </div>

            <div className={styles.reportDetails__people}>
              <Link
                className={styles.reportDetails__person}
                to={"/profile/" + report.reporter.id}
              >
                <UserRound
                  aria-hidden="true"
                  className={styles.reportDetails__personIcon}
                />
                <span className={styles.reportDetails__personContent}>
                  <small className={styles.reportDetails__personRole}>
                    Жалобщик
                  </small>
                  <strong className={styles.reportDetails__personName}>
                    {report.reporter.nickname}
                  </strong>
                </span>
              </Link>
              <Link
                className={[
                  styles.reportDetails__person,
                  styles.reportDetails__person_offender,
                ].join(" ")}
                to={"/profile/" + report.offender.id}
              >
                <AlertTriangle
                  aria-hidden="true"
                  className={styles.reportDetails__personIcon}
                />
                <span className={styles.reportDetails__personContent}>
                  <small className={styles.reportDetails__personRole}>
                    Автор сообщения
                  </small>
                  <strong className={styles.reportDetails__personName}>
                    {report.offender.nickname}
                  </strong>
                </span>
              </Link>
            </div>

            <div className={styles.reportDetails__message}>
              <span className={styles.reportDetails__sectionLabel}>
                Сообщение, на которое пожаловались
              </span>
              <blockquote className={styles.reportDetails__messageBody}>
                {report.message.body}
              </blockquote>
              <time className={styles.reportDetails__messageDate}>
                {formatDate(report.message.created_at)}
              </time>
            </div>

            <div className={styles.reportDetails__comment}>
              <span className={styles.reportDetails__sectionLabel}>
                Комментарий жалобщика
              </span>
              <p className={styles.reportDetails__commentText}>
                {report.comment?.trim() || "Комментарий не указан."}
              </p>
            </div>

            <div className={styles.reportDetails__assignment}>
              <span className={styles.reportDetails__sectionLabel}>
                Ответственный администратор
              </span>
              <strong className={styles.reportDetails__assignmentName}>
                {report.assignee?.nickname ?? "Жалоба ещё не назначена"}
              </strong>
              {report.assigned_at && (
                <span className={styles.reportDetails__assignmentDate}>
                  Назначена {formatDate(report.assigned_at)}
                </span>
              )}
            </div>

            {report.resolution_comment && (
              <div className={styles.reportDetails__resolution}>
                <span className={styles.reportDetails__sectionLabel}>
                  Решение администратора
                </span>
                <p className={styles.reportDetails__resolutionText}>
                  {report.resolution_comment}
                </p>
                <span className={styles.reportDetails__resolutionDate}>
                  Закрыта {formatDate(report.closed_at)}
                </span>
              </div>
            )}

            <div className={styles.reportDetails__actions}>
              <Button
                className={styles.reportDetails__action}
                color="transparent"
                onClick={() => setMessagesOpen((open) => !open)}
              >
                <MessageSquareText aria-hidden="true" size={17} />
                {messagesOpen ? "Скрыть переписку" : "Показать переписку"}
              </Button>

              {report.status === "open" && !report.assignee && (
                <Button
                  className={styles.reportDetails__action}
                  disabled={assignMutation.isPending}
                  onClick={() => assignMutation.mutate()}
                >
                  <ShieldCheck aria-hidden="true" size={17} />
                  {assignMutation.isPending
                    ? "Назначаем..."
                    : "Взять жалобу в работу"}
                </Button>
              )}

              {report.status === "open" && assignedToCurrentAdmin && (
                <>
                  <Button
                    className={styles.reportDetails__action}
                    color="danger"
                    onClick={() => {
                      decisionMutation.reset();
                      setDecision("reject");
                    }}
                  >
                    <XCircle aria-hidden="true" size={17} />
                    Отклонить
                  </Button>
                  <Button
                    className={styles.reportDetails__action}
                    onClick={() => {
                      decisionMutation.reset();
                      setDecision("resolve");
                    }}
                  >
                    <CheckCircle2 aria-hidden="true" size={17} />
                    Подтвердить
                  </Button>
                </>
              )}
            </div>

            {assignedToAnotherAdmin && report.status === "open" && (
              <p className={styles.reportDetails__notice}>
                Жалобу уже рассматривает {report.assignee?.nickname}. Решение
                может принять только назначенный администратор.
              </p>
            )}

            {assignMutation.isError && (
              <p className={styles.reportDetails__error} role="alert">
                {getAdminErrorMessage(
                  assignMutation.error,
                  "Не удалось взять жалобу в работу.",
                )}
              </p>
            )}

            {messagesOpen && (
              <AdminReportMessages
                reportId={report.id}
                reportedMessageId={report.message.id}
              />
            )}
          </div>
        )}
      </section>

      {decision && (
        <AdminReportDecision
          error={
            decisionMutation.isError
              ? getAdminErrorMessage(
                  decisionMutation.error,
                  "Не удалось сохранить решение.",
                )
              : undefined
          }
          isPending={decisionMutation.isPending}
          type={decision}
          onClose={() => {
            if (!decisionMutation.isPending) {
              setDecision(null);
              decisionMutation.reset();
            }
          }}
          onConfirm={(comment) =>
            decisionMutation.mutate({ comment, type: decision })
          }
        />
      )}
    </div>
  );
};
