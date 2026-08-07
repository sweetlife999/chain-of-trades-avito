import { memo, useState } from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  completeExchange,
  confirmExchange,
  declineExchange,
  getExchange,
} from "../../../../Api/exchanges/exchanges";
import type {
  TExchangeParticipant,
  TExchangeStatus,
} from "../../../../Api/exchanges/exchanges.types";
import { useAuthSelector } from "../../../../Hooks/useAuthDispatch";
import { ExchangeChat } from "../ExchangeChat/ExchangeChat";
import { ExchangeProgress } from "../ExchangeProgress/ExchangeProgress";
import { ConfirmationPopup } from "../../../UI/ConfirmationPopup/ConfirmationPopup";

const statusLabels: Record<TExchangeStatus, string> = {
  proposed: "Ждём подтверждения",
  confirmed: "Передача вещи в ПВЗ",
  completed: "Обмен завершён",
  cancelled: "Цепочка распалась",
};

const isParticipationConfirmed = (status: string) =>
  ["confirmed", "accepted"].includes(status.toLowerCase());

const isParticipationDeclined = (status: string) =>
  ["declined", "rejected"].includes(status.toLowerCase());

const isCurrentStageConfirmed = (
  participant: TExchangeParticipant,
  exchangeStatus: TExchangeStatus,
) => {
  if (exchangeStatus === "completed") {
    return true;
  }

  if (exchangeStatus === "proposed") {
    return isParticipationConfirmed(participant.status);
  }

  if (exchangeStatus === "confirmed") {
    return Boolean(participant.completion_confirmed_at);
  }

  return false;
};

const getParticipantStatus = (
  participant: TExchangeParticipant,
  exchangeStatus: TExchangeStatus,
) => {
  if (exchangeStatus === "completed") {
    return "Завершил";
  }

  if (isParticipationDeclined(participant.status)) {
    return "Отказался";
  }

  if (exchangeStatus === "confirmed") {
    return participant.completion_confirmed_at
      ? "Получил"
      : "Ожидает получения";
  }

  if (exchangeStatus === "proposed") {
    return isParticipationConfirmed(participant.status)
      ? "Подтвердил"
      : "Ожидает";
  }

  return "Обмен отменён";
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));

type TParticipantListProps = {
  participants: TExchangeParticipant[];
  exchangeStatus: TExchangeStatus;
  currentUserId?: string;
};

const ParticipantList = ({
  participants,
  exchangeStatus,
  currentUserId,
}: TParticipantListProps) => (
  <div className={styles.details__participants}>
    {participants.map((participant) => {
      const confirmed = isCurrentStageConfirmed(participant, exchangeStatus);

      const profilePath =
        participant.user.id === currentUserId
          ? "/profile"
          : `/profile/${participant.user.id}`;

      return (
        <div key={participant.user.id}>
          <Link className={styles.details__avatarLink} to={profilePath}>
            <span
              className={`${styles.details__avatar} ${
                confirmed ? styles.details__avatar_confirmed : ""
              }`}
            >
              {participant.user.photo_url ? (
                <img
                  alt={participant.user.nickname}
                  src={participant.user.photo_url}
                />
              ) : (
                participant.user.nickname.charAt(0).toUpperCase()
              )}
            </span>
          </Link>
          <Link className={styles.details__nameLink} to={profilePath}>
            <strong>{participant.user.nickname}</strong>
          </Link>
          <span>{getParticipantStatus(participant, exchangeStatus)}</span>
        </div>
      );
    })}
  </div>
);

const ExchangeDetailsComponent = () => {
  const { id = "" } = useParams();
  const { user } = useAuthSelector();
  const queryClient = useQueryClient();
  const [declinePopupOpen, setDeclinePopupOpen] = useState(false);

  const {
    data: exchange,
    isPending,
    isError,
  } = useQuery({
    queryKey: ["exchanges", id],
    queryFn: () => getExchange(id),
    enabled: Boolean(id),
  });

  const refreshExchange = () =>
    queryClient.invalidateQueries({ queryKey: ["exchanges"] });

  const confirmMutation = useMutation({
    mutationFn: () => confirmExchange(id),
    onSuccess: refreshExchange,
  });

  const declineMutation = useMutation({
    mutationFn: () => declineExchange(id),
    onSuccess: () => {
      setDeclinePopupOpen(false);
      refreshExchange();
    },
  });

  const completeMutation = useMutation({
    mutationFn: () => completeExchange(id),
    onSuccess: refreshExchange,
  });

  if (isPending) {
    return <p>Загрузка цепочки...</p>;
  }

  if (isError || !exchange) {
    return <p>Не удалось загрузить цепочку</p>;
  }

  const current = exchange.participants.find(
    ({ user: participant }) => participant.id === user?.id,
  );
  const referenceParticipant = current ?? exchange.participants[0];
  const isParticipant = Boolean(current);
  const participationConfirmed = current
    ? isParticipationConfirmed(current.status)
    : false;
  const receiptConfirmed = Boolean(current?.completion_confirmed_at);
  const actionPending =
    confirmMutation.isPending ||
    declineMutation.isPending ||
    completeMutation.isPending;
  const actionError = confirmMutation.isError || completeMutation.isError;
  const chatReadOnly =
    exchange.status === "completed" || exchange.status === "cancelled";
  const openDeclinePopup = () => {
    declineMutation.reset();
    setDeclinePopupOpen(true);
  };
  const closeDeclinePopup = () => {
    if (declineMutation.isPending) {
      return;
    }

    declineMutation.reset();
    setDeclinePopupOpen(false);
  };
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
            {exchange.participants.length} участника · создана{" "}
            {formatDate(exchange.created_at)}
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
            <ParticipantList
              currentUserId={user?.id}
              exchangeStatus={exchange.status}
              participants={exchange.participants}
            />
          </section>

          <aside className={styles.details__actions}>
            {isParticipant ? (
              <>
                <h2>Что нужно сделать</h2>
                <div className={styles.details__notice}>
                  Подтвердите участие. До подтверждения ваша вещь остаётся у вас.
                </div>
                <h3>Подтверждения участников</h3>
                <ParticipantList
                  currentUserId={user?.id}
                  exchangeStatus={exchange.status}
                  participants={exchange.participants}
                />
                <div className={styles.details__buttons}>
                  <button
                    className={styles.details__primaryButton}
                    disabled={actionPending || participationConfirmed}
                    onClick={() => confirmMutation.mutate()}
                    type="button"
                  >
                    {participationConfirmed
                      ? "Участие подтверждено"
                      : confirmMutation.isPending
                        ? "Подтверждаем..."
                        : "Подтвердить"}
                  </button>
                  <button
                    className={styles.details__dangerButton}
                    disabled={actionPending}
                    onClick={openDeclinePopup}
                    type="button"
                  >
                    {declineMutation.isPending
                      ? "Отказываемся..."
                      : "Отказаться"}
                  </button>
                </div>
                {actionError && (
                  <p className={styles.details__error}>
                    Не удалось выполнить действие. Повторите попытку.
                  </p>
                )}
              </>
            ) : (
              <>
                <h2>Хотите принять участие в этой цепочке?</h2>
                <h3>Подтверждения участников</h3>
                <ParticipantList
                  currentUserId={user?.id}
                  exchangeStatus={exchange.status}
                  participants={exchange.participants}
                />
                <Link className={styles.details__join} to="/myItems">
                  Предложить вещь
                </Link>
              </>
            )}
          </aside>
        </div>
      )}

      {exchange.status === "confirmed" && (
        <>
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
                <p>
                  Сотрудник примет вещь, сверит описание и отметит передачу.
                </p>
                <span>После сдачи статус обновится автоматически</span>
              </div>
            </div>
          </section>

          {isParticipant && (
            <section className={styles.details__receipt}>
              <div>
                <span>Этап получения</span>
                <h2>
                  {receiptConfirmed
                    ? "Получение подтверждено"
                    : "Вы получили вещь?"}
                </h2>
                <p>
                  {receiptConfirmed
                    ? "Ожидаем подтверждения остальных участников."
                    : "Подтвердите получение только после проверки вещи."}
                </p>
              </div>

              <div className={styles.details__participantProgress}>
                <h3>Получение участников</h3>
                <ParticipantList
                  currentUserId={user?.id}
                  exchangeStatus={exchange.status}
                  participants={exchange.participants}
                />
              </div>

              <div className={styles.details__buttons}>
                <button
                  className={styles.details__primaryButton}
                  disabled={actionPending || receiptConfirmed}
                  onClick={() => completeMutation.mutate()}
                  type="button"
                >
                  {receiptConfirmed
                    ? "Получение подтверждено"
                    : completeMutation.isPending
                      ? "Подтверждаем..."
                      : "Подтвердить получение"}
                </button>
                <button
                  className={styles.details__dangerButton}
                  disabled={actionPending || receiptConfirmed}
                  onClick={openDeclinePopup}
                  type="button"
                >
                  {declineMutation.isPending
                    ? "Отказываемся..."
                    : "Отказаться от обмена"}
                </button>
              </div>
              {actionError && (
                <p className={styles.details__error}>
                  Не удалось выполнить действие. Повторите попытку.
                </p>
              )}
            </section>
          )}
        </>
      )}

      {exchange.status === "completed" && (
        <section className={styles.details__result}>
          <span className={styles.details__resultIcon}>✓</span>
          <h2>Обмен завершён</h2>
          <p>Все участники получили свои вещи.</p>
          <ParticipantList
            currentUserId={user?.id}
            exchangeStatus={exchange.status}
            participants={exchange.participants}
          />
          <Link to="/exchanges">Вернуться к цепочкам</Link>
        </section>
      )}

      {exchange.status === "cancelled" && (
        <section className={styles.details__result}>
          <span
            className={`${styles.details__resultIcon} ${styles.details__resultIcon_error}`}
          >
            !
          </span>
          <h2>Один из участников отказался</h2>
          <p>Ваша вещь не передана и остаётся доступной.</p>
          <div className={styles.details__resultButtons}>
            <Link to="/">Искать новую цепочку</Link>
            <Link to="/myItems">Вернуть вещь в каталог</Link>
          </div>
        </section>
      )}

      {isParticipant && user && (
        <ExchangeChat
          currentUserId={user.id}
          exchangeId={exchange.id}
          readOnly={chatReadOnly}
        />
      )}

      {declinePopupOpen &&
        exchange.status !== "completed" &&
        exchange.status !== "cancelled" && (
          <ConfirmationPopup
            confirmLabel="Да, отказаться"
            description={
              exchange.status === "confirmed"
                ? "Подтверждённый обмен будет отменён. Отказ может увеличить счётчик сорванных обменов."
                : "После отказа эта цепочка будет отменена. Подтвердите действие, если точно не хотите участвовать."
            }
            error={
              declineMutation.isError
                ? "Не удалось отказаться от обмена. Повторите попытку."
                : undefined
            }
            isPending={declineMutation.isPending}
            title="Отказаться от обмена?"
            onClose={closeDeclinePopup}
            onConfirm={() => declineMutation.mutate()}
          />
        )}
    </section>
  );
};

export const ExchangeDetails = memo(ExchangeDetailsComponent);