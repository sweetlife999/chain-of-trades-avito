import { memo } from "react";

import { DashboardStats } from "../DashboardStats/DashboardStats";
import { PickupPoints } from "../PickupPoints/PickupPoints";
import { AdminReports } from "../AdminReports/AdminReports";
import { AdminExchangeDelivery } from "../AdminExchangeDelivery/AdminExchangeDelivery";
import styles from "./Styles.module.scss";

const AdminComponent = () => {
  return (
    <section className={styles.admin}>
      <header className={styles.admin__header}>
        <span className={styles.admin__eyebrow}>Администрирование</span>
        <h1 className={styles.admin__title}>Панель управления</h1>
        <p className={styles.admin__description}>
          Статистика сервиса, управление доставкой, пункты выдачи и очередь жалоб
        </p>
      </header>

      <DashboardStats />
      <AdminExchangeDelivery />
      <PickupPoints />
      <AdminReports />
    </section>
  );
};

export const Admin = memo(AdminComponent);
