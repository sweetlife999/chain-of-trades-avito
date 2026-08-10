import { memo } from "react";
import { MapPin } from "lucide-react";
import clsx from "clsx";

import styles from "./Styles.module.scss";
import type { TItem, TItemStatus } from "../../../Api/items/items.types";

type TProps = {
  className?: string;
  item: TItem;
};

const labels: Record<TItemStatus, string> = {
  available: "Доступно",
  reserved: "В цепочке",
  traded: "Обменяно",
  withdrawn: "Снято",
};

// Меток две, потому что в БД это два независимых поля: вещь может одновременно
// участвовать в цепочке и лежать в пункте выдачи.
const ItemStatusBadgeComponent = ({ className, item }: TProps) => (
  <span className={clsx(styles.badge, className)}>
    <span
      className={clsx(
        styles.badge__status,
        styles[`badge__status_${item.status}`],
      )}
    >
      {labels[item.status]}
    </span>

    {item.pickup_point && (
      <span className={styles.badge__pickup}>
        <MapPin aria-hidden="true" className={styles.badge__pickupIcon} />
        В ПВЗ
      </span>
    )}
  </span>
);

export const ItemStatusBadge = memo(ItemStatusBadgeComponent);
