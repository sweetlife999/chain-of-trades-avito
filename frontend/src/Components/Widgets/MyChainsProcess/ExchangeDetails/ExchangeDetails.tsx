import { memo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import { getExchange } from "../../../../Api/exchanges/exchanges";
import type {
  TExchangeParticipant,
  TExchangeStatus,
} from "../../../../Api/exchanges/exchanges.types";
import { useAuthSelector } from "../../../../Hooks/useAuthDispatch";
import { ExchangeProgress } from "../ExchangeProgress/ExchangeProgress";

const statusLabels: Record<TExchangeStatus, string> = {
  proposed: "Ждём подтверждения",
  confirmed: "Передача вещи в ПВЗ",
  completed: "Обмен завершён",
  cancelled: "Цепочка распалась",
};

const participantStatus = (status: string) => {
  const value = status.toLowerCase();
  if (["confirmed", "accepted"].includes(value)) {return "Подтвердил"};
  if (["declined", "rejected"].includes(value)) {return "Отказался"};
  return "Ожидает";
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));

const ParticipantList = ({ participants }: { participants: TExchangeParticipant[] }) => (
  <div className={styles.details__participants}>
    {participants.map((participant) => (
      <div key={participant.user.id}>
        <span className={styles.details__avatar}>
          {participant.user.nickname.charAt(0).toUpperCase()}
        </span>
        <strong>{participant.user.nickname}</strong>
        <span>{participantStatus(participant.status)}</span>
      </div>
    ))}
  </div>
);

const ExchangeDetailsComponent = () => {
  const { id = "" } = useParams();
  const { user } = useAuthSelector();
  const { data: exchange, isPending, isError } = useQuery({
    queryKey: ["exchanges", id],
    queryFn: () => getExchange(id),
    enabled: Boolean(id),
  });

  if (isPending) {return <p>Загрузка цепочки...</p>};
  if (isError || !exchange) {return <p>Не удалось загрузить цепочку</p>};

  const current = exchange.participants.find(
    ({ user: participant }) => participant.id === user?.id,
  );
  const referenceParticipant = current ?? exchange.participants[0];
  const isParticipant = Boolean(current);
  const title = referenceParticipant
    ? `${referenceParticipant.gives_item.title} → ${referenceParticipant.receives_item.title}`
    : "Цепочка обмена";

  return (
    <section className={styles.details}>
      <header className={styles.details__header}>
        <div>
          <Link to={isParticipant ? "/exchanges" : "/"}>
            ← {isParticipant ? "Мои цепочки" : "Обмены"}
          </Link>
          <h1>{title}</h1>
          <p>
            {exchange.participants.length} участника · создана {formatDate(exchange.created_at)}
          </p>
        </div>
        <span className={styles[`status_${exchange.status}`]}>
          {statusLabels[exchange.status]}
        </span>
      </header>

      <div className={styles.details__progress}>
        <ExchangeProgress status={exchange.status} />
      </div>

      {exchange.status === "proposed" && (
        <div className={styles.details__columns}>
          <section className={styles.details__scheme}>
            <h2>Кто и что получает</h2>
            {current && (
              <div className={styles.details__current}>
                <span>Вы</span>
                <strong>{current.gives_item.title}</strong>
                <small>Отдаёте эту вещь</small>
              </div>
            )}
            {current && (
              <div className={styles.details__receive}>
                <small>Вы получаете</small>
                <strong>{current.receives_item.title}</strong>
              </div>
            )}
            <ParticipantList participants={exchange.participants} />
          </section>

          <aside className={styles.details__actions}>
            {isParticipant ? (
              <>
                <h2>Что нужно сделать</h2>
                <div className={styles.details__notice}>
                  Подтвердите участие. До подтверждения ваша вещь остаётся у вас.
                </div>
                <h3>Подтверждения участников</h3>
                <ParticipantList participants={exchange.participants} />
                <div className={styles.details__buttons}>
                  <button disabled type="button">Подтвердить</button>
                  <button disabled type="button">Отказаться</button>
                </div>
              </>
            ) : (
              <>
                <h2>Хотите принять участие в этой цепочке?</h2>
                <h3>Подтверждения участников</h3>
                <ParticipantList participants={exchange.participants} />
                <Link className={styles.details__join} to="/myItems">
                  Предложить вещь
                </Link>
              </>
            )}
          </aside>
        </div>
      )}

      {exchange.status === "confirmed" && (
        <section className={styles.details__pvz}>
          <h2>Сдайте вещь в пункт выдачи</h2>
          <p>ПВЗ проверит товар и зафиксирует его состояние.</p>
          <div className={styles.details__address}>
            <strong>ПВЗ на Невском проспекте</strong>
            <span>Санкт-Петербург, Невский проспект, 88</span>
            <span>Сегодня до 22:00 · 1,2 км от вас</span>
          </div>
          <div className={styles.details__qrRow}>
            <div className={styles.details__qr}>QR</div>
            <div>
              <h3>Покажите QR-код сотруднику</h3>
              <p>Сотрудник примет вещь, сверит описание и отметит передачу.</p>
              <span>После сдачи статус обновится автоматически</span>
            </div>
          </div>
        </section>
      )}

      {exchange.status === "completed" && (
        <section className={styles.details__result}>
          <span className={styles.details__resultIcon}>✓</span>
          <h2>Обмен завершён</h2>
          <p>Все участники получили свои вещи.</p>
          <Link to="/myChains">Вернуться к цепочкам</Link>
        </section>
      )}

      {exchange.status === "cancelled" && (
        <section className={styles.details__result}>
          <span className={`${styles.details__resultIcon} ${styles.details__resultIcon_error}`}>!</span>
          <h2>Один из участников отказался</h2>
          <p>Ваша вещь не передана и остаётся доступной.</p>
          <div className={styles.details__resultButtons}>
            <Link to="/">Искать новую цепочку</Link>
            <Link to="/myItems">Вернуть вещь в каталог</Link>
          </div>
        </section>
      )}
    </section>
  );
};

export const ExchangeDetails = memo(ExchangeDetailsComponent);
