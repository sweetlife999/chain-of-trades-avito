import { memo } from "react";

import { DashboardStats } from "../DashboardStats/DashboardStats";
import { PickupPoints } from "../PickupPoints/PickupPoints";
import styles from "./Styles.module.scss";

const AdminComponent = () => {
  return (
    <section className={styles.admin}>
      <header className={styles.admin__header}>
        <span className={styles.admin__eyebrow}>Администрирование</span>
        <h1>Панель управления</h1>
        <p>Статистика сервиса и управление пунктами выдачи</p>
      </header>

      <DashboardStats />
      <PickupPoints />
    </section>
  );
};

export const Admin = memo(AdminComponent);
