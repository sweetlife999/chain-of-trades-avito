import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Filter,
  MessageSquareWarning,
  UserCheck,
} from "lucide-react";

import { getAdminReports } from "../../../../Api/admin/admin";
import type {
  TAdminReportReason,
  TAdminReportStatus,
} from "../../../../Api/admin/admin.types";
import { useAuthSelector } from "../../../../Hooks/useAuthDispatch";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import { AdminReportDetails } from "../AdminReportDetails/AdminReportDetails";
import styles from "./Styles.module.scss";

const PAGE_SIZE = 10;

type TStatusFilter = "all" | TAdminReportStatus;
type TReasonFilter = "all" | TAdminReportReason;
type TAssigneeFilter = "all" | "mine";

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

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));

export const AdminReports = () => {
  const { user } = useAuthSelector();
  const [page, setPage] = useState(0);
  const [status, setStatus] = useState<TStatusFilter>("open");
  const [reason, setReason] = useState<TReasonFilter>("all");
  const [assignee, setAssignee] = useState<TAssigneeFilter>("all");
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null);
  const offset = page * PAGE_SIZE;

  const reportsQuery = useQuery({
    queryKey: [
      "admin",
      "reports",
      "list",
      { assignee, offset, reason, status },
    ],
    queryFn: () =>
      getAdminReports({
        assignee_id: assignee === "mine" ? user?.id : undefined,
        limit: PAGE_SIZE,
        offset,
        reason: reason === "all" ? undefined : reason,
        status: status === "all" ? undefined : status,
      }),
    enabled: Boolean(user),
    placeholderData: (previousData) => previousData,
    retry: false,
  });

  if (!user) {
    return null;
  }

  const reports = reportsQuery.data?.reports ?? [];
  const total = reportsQuery.data?.pagination.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const rangeStart = total === 0 ? 0 : offset + 1;
  const rangeEnd = Math.min(offset + reports.length, total);

  const resetPage = () => setPage(0);

  return (
    <section className={styles.reports} aria-labelledby="admin-reports-title">
      <div className={styles.reports__header}>
        <div className={styles.reports__heading}>
          <span className={styles.reports__icon} aria-hidden="true">
            <MessageSquareWarning className={styles.reports__iconImage} />
          </span>
          <div className={styles.reports__headingText}>
            <h2 className={styles.reports__title} id="admin-reports-title">
              Жалобы пользователей
            </h2>
            <p className={styles.reports__description}>
              Проверяйте сообщения, изучайте переписку и фиксируйте решение.
            </p>
          </div>
        </div>
        <span className={styles.reports__total}>Всего: {total}</span>
      </div>

      <div className={styles.reports__filters}>
        <span className={styles.reports__filtersLabel}>
          <Filter aria-hidden="true" className={styles.reports__filterIcon} />
          Фильтры
        </span>

        <label className={styles.reports__filter}>
          <span className={styles.reports__filterLabel}>Статус</span>
          <select
            className={styles.reports__select}
            onChange={(event) => {
              setStatus(event.target.value as TStatusFilter);
              resetPage();
            }}
            value={status}
          >
            <option value="all">Все статусы</option>
            <option value="open">Открытые</option>
            <option value="resolved">Подтверждённые</option>
            <option value="rejected">Отклонённые</option>
          </select>
        </label>

        <label className={styles.reports__filter}>
          <span className={styles.reports__filterLabel}>Причина</span>
          <select
            className={styles.reports__select}
            onChange={(event) => {
              setReason(event.target.value as TReasonFilter);
              resetPage();
            }}
            value={reason}
          >
            <option value="all">Все причины</option>
            <option value="spam">Спам</option>
            <option value="abuse">Оскорбления</option>
            <option value="other">Другое</option>
          </select>
        </label>

        <label className={styles.reports__filter}>
          <span className={styles.reports__filterLabel}>Назначение</span>
          <select
            className={styles.reports__select}
            onChange={(event) => {
              setAssignee(event.target.value as TAssigneeFilter);
              resetPage();
            }}
            value={assignee}
          >
            <option value="all">Все администраторы</option>
            <option value="mine">Назначенные мне</option>
          </select>
        </label>
      </div>

      {reportsQuery.isPending && (
        <div className={styles.reports__state}>
          <Loader text="Загружаем очередь жалоб..." />
        </div>
      )}

      {reportsQuery.isError && (
        <div className={styles.reports__state} role="alert">
          <AlertTriangle
            aria-hidden="true"
            className={styles.reports__stateIcon}
          />
          <strong className={styles.reports__stateTitle}>
            Не удалось загрузить жалобы
          </strong>
          <p className={styles.reports__stateDescription}>
            Проверьте соединение и права администратора.
          </p>
          <Button
            color="light"
            size="s"
            onClick={() => void reportsQuery.refetch()}
          >
            Повторить
          </Button>
        </div>
      )}

      {!reportsQuery.isPending &&
        !reportsQuery.isError &&
        reports.length === 0 && (
          <div className={styles.reports__state}>
            <MessageSquareWarning
              aria-hidden="true"
              className={styles.reports__stateIcon}
            />
            <strong className={styles.reports__stateTitle}>
              Жалоб по выбранным фильтрам нет
            </strong>
            <p className={styles.reports__stateDescription}>
              Измените фильтры или вернитесь к очереди позже.
            </p>
          </div>
        )}

      {reports.length > 0 && (
        <div
          aria-busy={reportsQuery.isFetching}
          className={styles.reports__list}
        >
          {reports.map((report) => (
            <button
              className={styles.reports__card}
              key={report.id}
              onClick={() => setSelectedReportId(report.id)}
              type="button"
            >
              <div className={styles.reports__cardHeader}>
                <span
                  className={[
                    styles.reports__status,
                    styles["reports__status_" + report.status],
                  ].join(" ")}
                >
                  {statusLabels[report.status]}
                </span>
                <span className={styles.reports__reason}>
                  {reasonLabels[report.reason]}
                </span>
                <time className={styles.reports__date}>
                  {formatDate(report.created_at)}
                </time>
              </div>

              <blockquote className={styles.reports__message}>
                {report.message.body}
              </blockquote>

              <div className={styles.reports__people}>
                <span className={styles.reports__person}>
                  Жалобщик: <strong>{report.reporter.nickname}</strong>
                </span>
                <span className={styles.reports__person}>
                  Нарушитель: <strong>{report.offender.nickname}</strong>
                </span>
                <span className={styles.reports__assignee}>
                  <UserCheck aria-hidden="true" size={15} />
                  {report.assignee?.nickname ?? "Не назначена"}
                </span>
              </div>
            </button>
          ))}
        </div>
      )}

      {!reportsQuery.isError && total > PAGE_SIZE && (
        <nav aria-label="Пагинация жалоб" className={styles.reports__pagination}>
          <span className={styles.reports__paginationSummary}>
            Показано {rangeStart}–{rangeEnd} из {total}
          </span>
          <div className={styles.reports__paginationControls}>
            <Button
              color="transparent"
              disabled={page === 0 || reportsQuery.isFetching}
              size="s"
              onClick={() => setPage((currentPage) => currentPage - 1)}
            >
              <ChevronLeft aria-hidden="true" size={16} />
              Назад
            </Button>
            <span className={styles.reports__paginationCurrent}>
              {page + 1} / {totalPages}
            </span>
            <Button
              color="transparent"
              disabled={page + 1 >= totalPages || reportsQuery.isFetching}
              size="s"
              onClick={() => setPage((currentPage) => currentPage + 1)}
            >
              Далее
              <ChevronRight aria-hidden="true" size={16} />
            </Button>
          </div>
        </nav>
      )}

      {selectedReportId && (
        <AdminReportDetails
          currentAdminId={user.id}
          reportId={selectedReportId}
          onClose={() => setSelectedReportId(null)}
        />
      )}
    </section>
  );
};
