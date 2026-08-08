import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeftRight,
  MapPin,
  Package,
  RefreshCw,
  Users,
  type LucideIcon,
} from "lucide-react";

import { getAdminDashboard } from "../../../../Api/admin/admin";
import type { TDashboard } from "../../../../Api/admin/admin.types";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import styles from "./Styles.module.scss";

type TMainStat = {
  label: string;
  value: number;
  Icon: LucideIcon;
  color: "blue" | "green" | "orange" | "violet";
};

type TStatusRow = {
  label: string;
  value: number;
};

const getMainStats = (dashboard: TDashboard): TMainStat[] => [
  {
    label: "Пользователи",
    value: dashboard.users_total,
    Icon: Users,
    color: "blue",
  },
  {
    label: "Пункты выдачи",
    value: dashboard.pickup_points_total,
    Icon: MapPin,
    color: "green",
  },
  {
    label: "Объявления",
    value: dashboard.items.total,
    Icon: Package,
    color: "orange",
  },
  {
    label: "Обмены",
    value: dashboard.exchanges.total,
    Icon: ArrowLeftRight,
    color: "violet",
  },
];

const StatusList = ({
  title,
  total,
  rows,
}: {
  title: string;
  total: number;
  rows: TStatusRow[];
}) => (
  <article className={styles.dashboard__detailsCard}>
    <header>
      <h3>{title}</h3>
      <strong>{total}</strong>
    </header>
    <dl>
      {rows.map(({ label, value }) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  </article>
);

export const DashboardStats = () => {
  const dashboardQuery = useQuery({
    queryKey: ["admin", "dashboard"],
    queryFn: getAdminDashboard,
  });

  return (
    <section className={styles.dashboard} aria-labelledby="admin-dashboard-title">
      <div className={styles.admin__sectionHeader}>
        <div>
          <h2 id="admin-dashboard-title">Статистика</h2>
          <p>Актуальные показатели платформы</p>
        </div>
        {!dashboardQuery.isPending && (
          <button
            className={styles.admin__refresh}
            type="button"
            disabled={dashboardQuery.isFetching}
            onClick={() => {
              void dashboardQuery.refetch();
            }}
          >
            <RefreshCw aria-hidden="true" />
            {dashboardQuery.isFetching ? "Обновляем..." : "Обновить"}
          </button>
        )}
      </div>

      {dashboardQuery.isPending && (
        <div className={styles.admin__state}>
          <Loader text="Загружаем статистику..." />
        </div>
      )}

      {dashboardQuery.isError && (
        <div className={styles.admin__state} role="alert">
          <h3>Не удалось загрузить статистику</h3>
          <p>Проверьте соединение с сервером и повторите запрос.</p>
          <Button
            color="light"
            size="s"
            onClick={() => {
              void dashboardQuery.refetch();
            }}
          >
            Повторить
          </Button>
        </div>
      )}

      {dashboardQuery.data && (
        <>
          <div className={styles.dashboard__mainGrid}>
            {getMainStats(dashboardQuery.data).map(
              ({ label, value, Icon, color }) => (
                <article className={styles.dashboard__mainCard} key={label}>
                  <span
                    className={`${styles.dashboard__mainIcon} ${
                      styles[`dashboard__mainIcon_${color}`]
                    }`}
                    aria-hidden="true"
                  >
                    <Icon />
                  </span>
                  <div>
                    <span>{label}</span>
                    <strong>{value.toLocaleString("ru-RU")}</strong>
                  </div>
                </article>
              ),
            )}
          </div>

          <div className={styles.dashboard__detailsGrid}>
            <StatusList
              title="Объявления по статусам"
              total={dashboardQuery.data.items.total}
              rows={[
                { label: "Доступны", value: dashboardQuery.data.items.available },
                { label: "Зарезервированы", value: dashboardQuery.data.items.reserved },
                { label: "Обменяны", value: dashboardQuery.data.items.traded },
                { label: "Сняты", value: dashboardQuery.data.items.withdrawn },
              ]}
            />
            <StatusList
              title="Обмены по статусам"
              total={dashboardQuery.data.exchanges.total}
              rows={[
                { label: "Предложены", value: dashboardQuery.data.exchanges.proposed },
                { label: "Подтверждены", value: dashboardQuery.data.exchanges.confirmed },
                { label: "Завершены", value: dashboardQuery.data.exchanges.completed },
                { label: "Отменены", value: dashboardQuery.data.exchanges.cancelled },
              ]}
            />
          </div>
        </>
      )}
    </section>
  );
};
