import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ChevronLeft,
  ChevronRight,
  Filter,
  History,
  RefreshCw,
} from "lucide-react";

import { getAdminAuditLog } from "../../../../Api/admin/admin";
import type { TAdminAuditLogParams } from "../../../../Api/admin/admin.types";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import styles from "./Styles.module.scss";

const PAGE_SIZE = 20;

type TAuditFilters = {
  adminId: string;
  action: string;
  from: string;
  to: string;
};

const EMPTY_FILTERS: TAuditFilters = {
  adminId: "",
  action: "",
  from: "",
  to: "",
};

const toRfc3339 = (value: string) =>
  value ? new Date(value).toISOString() : undefined;

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));

const formatMetadata = (metadata: Record<string, unknown>) => {
  const entries = Object.entries(metadata);

  if (entries.length === 0) {
    return "Нет дополнительных данных";
  }

  return entries
    .map(([key, value]) => `${key}: ${JSON.stringify(value)}`)
    .join(" · ");
};

export const AdminAuditLog = () => {
  const [page, setPage] = useState(0);
  const [draftFilters, setDraftFilters] = useState<TAuditFilters>(EMPTY_FILTERS);
  const [filters, setFilters] = useState<TAuditFilters>(EMPTY_FILTERS);
  const offset = page * PAGE_SIZE;

  const params = useMemo<TAdminAuditLogParams>(
    () => ({
      admin_id: filters.adminId.trim() || undefined,
      action: filters.action.trim() || undefined,
      from: toRfc3339(filters.from),
      to: toRfc3339(filters.to),
      limit: PAGE_SIZE,
      offset,
    }),
    [filters, offset],
  );

  const auditQuery = useQuery({
    queryKey: ["admin", "audit-log", params],
    queryFn: () => getAdminAuditLog(params),
    placeholderData: (previousData) => previousData,
    retry: false,
  });

  const entries = auditQuery.data?.entries ?? [];
  const total = auditQuery.data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const rangeStart = total === 0 ? 0 : offset + 1;
  const rangeEnd = Math.min(offset + entries.length, total);

  const applyFilters = () => {
    setPage(0);
    setFilters(draftFilters);
  };

  const resetFilters = () => {
    setPage(0);
    setDraftFilters(EMPTY_FILTERS);
    setFilters(EMPTY_FILTERS);
  };

  return (
    <section className={styles.audit} aria-labelledby="admin-audit-title">
      <div className={styles.audit__sectionHeader}>
        <div className={styles.audit__heading}>
          <span className={styles.audit__icon} aria-hidden="true">
            <History size={22} />
          </span>
          <div>
            <h2 className={styles.audit__title} id="admin-audit-title">
              История действий администраторов
            </h2>
            <p className={styles.audit__description}>
              Журнал административных действий с фильтрами по администратору,
              действию и периоду.
            </p>
          </div>
        </div>

        {!auditQuery.isPending && (
          <Button
            color="transparent"
            disabled={auditQuery.isFetching}
            size="s"
            type="button"
            onClick={() => void auditQuery.refetch()}
          >
            <RefreshCw aria-hidden="true" size={16} />
            {auditQuery.isFetching ? "Обновляем..." : "Обновить"}
          </Button>
        )}
      </div>

      <div className={styles.audit__filters}>
        <span className={styles.audit__filtersTitle}>
          <Filter aria-hidden="true" size={16} />
          Фильтры
        </span>

        <label className={styles.audit__filter}>
          <span className={styles.audit__filterLabel}>ID администратора</span>
          <input
            className={styles.audit__input}
            placeholder="UUID администратора"
            type="text"
            value={draftFilters.adminId}
            onChange={(event) =>
              setDraftFilters((current) => ({
                ...current,
                adminId: event.target.value,
              }))
            }
          />
        </label>

        <label className={styles.audit__filter}>
          <span className={styles.audit__filterLabel}>Действие</span>
          <input
            className={styles.audit__input}
            placeholder="Тип действия"
            type="text"
            value={draftFilters.action}
            onChange={(event) =>
              setDraftFilters((current) => ({
                ...current,
                action: event.target.value,
              }))
            }
          />
        </label>

        <label className={styles.audit__filter}>
          <span className={styles.audit__filterLabel}>С</span>
          <input
            className={styles.audit__input}
            type="datetime-local"
            value={draftFilters.from}
            onChange={(event) =>
              setDraftFilters((current) => ({
                ...current,
                from: event.target.value,
              }))
            }
          />
        </label>

        <label className={styles.audit__filter}>
          <span className={styles.audit__filterLabel}>По</span>
          <input
            className={styles.audit__input}
            type="datetime-local"
            value={draftFilters.to}
            onChange={(event) =>
              setDraftFilters((current) => ({
                ...current,
                to: event.target.value,
              }))
            }
          />
        </label>

        <div className={styles.audit__filterActions}>
          <Button color="green" size="s" type="button" onClick={applyFilters}>
            Применить
          </Button>
          <Button
            color="transparent"
            size="s"
            type="button"
            onClick={resetFilters}
          >
            Сбросить
          </Button>
        </div>
      </div>

      {auditQuery.isPending && (
        <div className={styles.audit__state}>
          <Loader text="Загружаем историю действий..." />
        </div>
      )}

      {auditQuery.isError && (
        <div className={styles.audit__state} role="alert">
          <strong className={styles.audit__stateTitle}>
            Не удалось загрузить журнал
          </strong>
          <p className={styles.audit__stateDescription}>
            Проверьте фильтры, соединение и права администратора.
          </p>
          <Button
            color="light"
            size="s"
            type="button"
            onClick={() => void auditQuery.refetch()}
          >
            Повторить
          </Button>
        </div>
      )}

      {!auditQuery.isPending && !auditQuery.isError && entries.length === 0 && (
        <div className={styles.audit__state}>
          <History aria-hidden="true" className={styles.audit__stateIcon} />
          <strong className={styles.audit__stateTitle}>Записей не найдено</strong>
          <p className={styles.audit__stateDescription}>
            Измените фильтры или дождитесь новых действий администраторов.
          </p>
        </div>
      )}

      {!auditQuery.isError && entries.length > 0 && (
        <div aria-busy={auditQuery.isFetching} className={styles.audit__list}>
          {entries.map((entry) => (
            <article className={styles.audit__entry} key={entry.id}>
              <div className={styles.audit__entryTop}>
                <strong className={styles.audit__action}>{entry.action}</strong>
                <time className={styles.audit__date} dateTime={entry.created_at}>
                  {formatDate(entry.created_at)}
                </time>
              </div>

              <dl className={styles.audit__details}>
                <div className={styles.audit__detail}>
                  <dt className={styles.audit__detailTerm}>Администратор</dt>
                  <dd className={styles.audit__detailValue}>{entry.admin_id}</dd>
                </div>
                <div className={styles.audit__detail}>
                  <dt className={styles.audit__detailTerm}>Объект</dt>
                  <dd className={styles.audit__detailValue}>
                    {entry.target_type}: {entry.target_id}
                  </dd>
                </div>
              </dl>

              <p className={styles.audit__metadata}>
                {formatMetadata(entry.metadata)}
              </p>
            </article>
          ))}
        </div>
      )}

      {!auditQuery.isError && total > PAGE_SIZE && (
        <nav aria-label="Пагинация истории действий" className={styles.pagination}>
          <span className={styles.pagination__summary}>
            Показано {rangeStart}–{rangeEnd} из {total}
          </span>
          <div className={styles.pagination__controls}>
            <Button
              color="transparent"
              disabled={page === 0 || auditQuery.isFetching}
              size="s"
              type="button"
              onClick={() => setPage((currentPage) => currentPage - 1)}
            >
              <ChevronLeft aria-hidden="true" size={16} />
              Назад
            </Button>
            <span className={styles.pagination__current}>
              {page + 1} / {totalPages}
            </span>
            <Button
              color="transparent"
              disabled={page + 1 >= totalPages || auditQuery.isFetching}
              size="s"
              type="button"
              onClick={() => setPage((currentPage) => currentPage + 1)}
            >
              Далее
              <ChevronRight aria-hidden="true" size={16} />
            </Button>
          </div>
        </nav>
      )}
    </section>
  );
};
