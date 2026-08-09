import { CheckCircle2, Clock3, PackageCheck } from "lucide-react";

import type {
  TExchangeParticipant,
  TExchangeStatus,
} from "../../../../Api/exchanges/exchanges.types";
import { PickupPointSelector } from "../../PickupPointSelector/PickupPointSelector";
import styles from "./Styles.module.scss";

type TProps = {
  currentUserId: string;
  exchangeStatus: Extract<TExchangeStatus, "confirmed" | "delivering">;
  participants: TExchangeParticipant[];
};

export const ExchangePickupStage = ({
  currentUserId,
  exchangeStatus,
  participants,
}: TProps) => {
  const current = participants.find(
    (participant) => participant.user.id === currentUserId,
  );
  const deliveredCount = participants.filter(
    (participant) => participant.gives_item.pickup_point,
  ).length;

  if (!current) {
    return null;
  }

  const isDelivering = exchangeStatus === "delivering";

  return (
    <section className={styles.pickupStage}>
      <div className={styles.pickupStage__header}>
        <span className={styles.pickupStage__icon} aria-hidden="true">
          <PackageCheck className={styles.pickupStage__iconImage} />
        </span>
        <div className={styles.pickupStage__heading}>
          <span className={styles.pickupStage__eyebrow}>
            {isDelivering ? "Этап доставки" : "Этап передачи в ПВЗ"}
          </span>
          <h2 className={styles.pickupStage__title}>
            {isDelivering
              ? "Все вещи приняты пунктами выдачи"
              : `Отнесите «${current.gives_item.title}» в удобный пункт`}
          </h2>
          <p className={styles.pickupStage__description}>
            {isDelivering
              ? "Вещи доставляются между пунктами. Мы покажем здесь, когда вашу можно будет забрать."
              : "Когда все вещи окажутся в пунктах выдачи, начнётся их доставка получателям."}
          </p>
        </div>
      </div>

      <PickupPointSelector
        currentPickupPoint={current.gives_item.pickup_point}
        itemId={current.gives_item.id}
      />

      <div className={styles.pickupStage__progress}>
        <div className={styles.pickupStage__progressHeader}>
          <h3 className={styles.pickupStage__progressTitle}>
            {isDelivering ? "Все вещи находятся в ПВЗ" : "Передача вещей участниками"}
          </h3>
          <span className={styles.pickupStage__progressCount}>
            {deliveredCount} из {participants.length}
          </span>
        </div>

        <div className={styles.pickupStage__participants}>
          {participants.map((participant) => {
            const pickupPoint = participant.gives_item.pickup_point;

            return (
              <div
                className={styles.pickupStage__participant}
                key={participant.user.id}
              >
                {pickupPoint ? (
                  <CheckCircle2
                    aria-hidden="true"
                    className={styles.pickupStage__participantIcon_done}
                  />
                ) : (
                  <Clock3
                    aria-hidden="true"
                    className={styles.pickupStage__participantIcon}
                  />
                )}
                <div className={styles.pickupStage__participantContent}>
                  <strong className={styles.pickupStage__participantName}>
                    {participant.user.id === currentUserId
                      ? "Вы"
                      : participant.user.nickname}
                  </strong>
                  <span className={styles.pickupStage__participantItem}>
                    {participant.gives_item.title}
                  </span>
                  <span className={styles.pickupStage__participantStatus}>
                    {pickupPoint
                      ? "Передано: " + pickupPoint.name
                      : "Пока находится у владельца"}
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {current.receives_item.pickup_point && (
        <div className={styles.pickupStage__receiving}>
          <span className={styles.pickupStage__receivingLabel}>
            Вещь, которую вы получите
          </span>
          <strong className={styles.pickupStage__receivingTitle}>
            {current.receives_item.title}
          </strong>
          <span className={styles.pickupStage__receivingAddress}>
            Сейчас в ПВЗ: {current.receives_item.pickup_point.name},{" "}
            {current.receives_item.pickup_point.address}
          </span>
        </div>
      )}
    </section>
  );
};
