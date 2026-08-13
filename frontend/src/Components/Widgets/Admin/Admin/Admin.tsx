import { memo } from "react";
import { useSearchParams } from "react-router-dom";

import { DashboardStats } from "../DashboardStats/DashboardStats";
import { PickupPoints } from "../PickupPoints/PickupPoints";
import { AdminReports } from "../AdminReports/AdminReports";
import { AdminExchangeDelivery } from "../AdminExchangeDelivery/AdminExchangeDelivery";
import { AdminAuditLog } from "../AdminAuditLog/AdminAuditLog";
import { AdminSupport } from "../AdminSupport/AdminSupport";
import { AdminAntiscam } from "../AdminAntiscam/AdminAntiscam";
import styles from "./Styles.module.scss";

type TAdminSection =
  | "statistics"
  | "delivery"
  | "pickup-points"
  | "reports"
  | "audit"
  | "support"
  | "antiscam";

const sections: Array<{ id: TAdminSection; label: string }> = [
  { id: "statistics", label: "Статистика" },
  { id: "delivery", label: "Завершение доставки" },
  { id: "pickup-points", label: "Пункты выдачи" },
  { id: "reports", label: "Жалобы пользователей" },
  { id: "antiscam", label: "AI-антискам" },
  { id: "support", label: "Поддержка" },
  { id: "audit", label: "История действий админов" },
];

const AdminComponent = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedSection = searchParams.get("section") as TAdminSection | null;
  const activeSection = sections.some(({ id }) => id === requestedSection)
    ? requestedSection as TAdminSection
    : "statistics";
  const selectSection = (section: TAdminSection) => {
    const next = new URLSearchParams(searchParams);
    next.set("section", section);
    if (section !== "support") {
      next.delete("thread");
    }
    setSearchParams(next);
  };

  return (
    <section className={styles.admin}>
      <div className={styles.admin__header}>
        <span className={styles.admin__eyebrow}>Администрирование</span>
        <h1 className={styles.admin__title}>Панель управления</h1>
        <p className={styles.admin__description}>
          Статистика сервиса, управление доставкой, пункты выдачи, жалобы и
          журнал действий администраторов.
        </p>
      </div>

      <nav className={styles.admin__tabs} aria-label="Разделы админ-панели">
        {sections.map((section) => (
          <button
            aria-current={activeSection === section.id ? "page" : undefined}
            className={`${styles.admin__tab} ${
              activeSection === section.id ? styles.admin__tab_active : ""
            }`}
            key={section.id}
            type="button"
            onClick={() => selectSection(section.id)}
          >
            {section.label}
          </button>
        ))}
      </nav>

      <div className={styles.admin__content}>
        {activeSection === "statistics" && <DashboardStats />}
        {activeSection === "delivery" && <AdminExchangeDelivery />}
        {activeSection === "pickup-points" && <PickupPoints />}
        {activeSection === "reports" && <AdminReports />}
        {activeSection === "antiscam" && <AdminAntiscam />}
        {activeSection === "support" && <AdminSupport />}
        {activeSection === "audit" && <AdminAuditLog />}
      </div>
    </section>
  );
};

export const Admin = memo(AdminComponent);
