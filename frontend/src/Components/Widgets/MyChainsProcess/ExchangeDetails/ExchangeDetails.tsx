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
import { CancelExchangeButton } from "../../Admin/CancelExchangeButton/CancelExchangeButton";
import { ExchangePickupStage } from "../ExchangePickupStage/ExchangePickupStage";
import { ExchangeRating } from "../ExchangeRating/ExchangeRating";

const statusLabels: Record<TExchangeStatus, string> = {
  proposed: "Ждём подтверждения",
  confirmed: "Передача вещи в ПВЗ",
  delivering: "Доставка между ПВЗ",
  delivered: "Вещь ожидает получения",
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

  if (["confirmed", "delivering"].includes(exchangeStatus)) {
    return Boolean(participant.gives_item.pickup_point);
  }

  if (exchangeStatus === "delivered") {
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
    return participant.gives_item.pickup_point
      ? "Передал в ПВЗ"
      : "Ожидает передачи";
  }

  if (exchangeStatus === "delivering") {
    return participant.gives_item.pickup_point
      ? "Вещь в пути"
      : "Ожидает передачи";
  }

  if (exchangeStatus === "delivered") {
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
        <div className={styles.details__participant} key={participant.user.id}>
          <Link className={styles.details__avatarLink} to={profilePath}>
            <span
              className={`${styles.details__avatar} ${
                confirmed ? styles.details__avatar_confirmed : ""
              }`}
            >
              {participant.user.photo_url ? (
                <img
                  alt={participant.user.nickname}
                  className={styles.details__avatarImage}
                  src={participant.user.photo_url}
                />
              ) : (
                participant.user.nickname.charAt(0).toUpperCase()
              )}
            </span>
          </Link>
          <Link className={styles.details__nameLink} to={profilePath}>
            <strong className={styles.details__participantName}>
              {participant.user.nickname}
            </strong>
          </Link>
          <span className={styles.details__participantStatus}>
            {getParticipantStatus(participant, exchangeStatus)}
          </span>
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
  const [cancelledByAdmin, setCancelledByAdmin] = useState(false);

  const {
    data: exchange,
    isPending,
    isError,
  } = useQuery({
    queryKey: ["exchanges", id],
    queryFn: () => getExchange(id),
    enabled: Boolean(id),
    retry: false,
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
    return <p className={styles.details__state}>Загрузка цепочки...</p>;
  }

  if (isError || !exchange) {
    return (
      <p className={`${styles.details__state} ${styles.details__state_error}`}>
        Не удалось загрузить цепочку
      </p>
    );
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
  const isAdmin = Boolean(user?.is_admin);
  const adminCanCancel =
    isAdmin && ["proposed", "confirmed"].includes(exchange.status);
  // Кого оценивать, решил сервер — здесь только достаём его ник для подписи.
  const ratedPartnerNickname =
    exchange.participants.find(
      ({ user: participant }) => participant.id === exchange.rating?.rated_user_id,
    )?.user.nickname ?? "участника";
  const returnTo = isParticipant ? "/exchanges" : "/feed";
  const returnLabel = isParticipant ? "Мои цепочки" : "Обмены";

  return (
    <section className={styles.details}>
      <div className={styles.details__header}>
        <div className={styles.details__heading}>
          <Link className={styles.details__back} to={returnTo}>
            ← {returnLabel}
          </Link>
          <h1 className={styles.details__title}>{title}</h1>
          <p className={styles.details__meta}>
            {exchange.participants.length} участника · создана{" "}
            {formatDate(exchange.created_at)}
          </p>
        </div>
        <div className={styles.details__headerActions}>
          <span
            className={`${styles.details__status} ${styles[`details__status_${exchange.status}`]}`}
          >
            {statusLabels[exchange.status]}
          </span>
          {adminCanCancel && (
            <CancelExchangeButton
              exchangeId={exchange.id}
              onCancelled={() => {
                setCancelledByAdmin(true);
              }}
            />
          )}
        </div>
      </div>

      <div className={styles.details__progress}>
        <ExchangeProgress status={exchange.status} />
      </div>

      {exchange.status === "proposed" && (
        <div className={styles.details__columns}>
          <section className={styles.details__scheme}>
            <h2 className={styles.details__sectionTitle}>
              Кто и что получает
            </h2>
            {current && (
              <div className={styles.details__current}>
                <span className={styles.details__currentAvatar}>Вы</span>
                <strong className={styles.details__itemTitle}>
                  {current.gives_item.title}
                </strong>
                <small className={styles.details__itemHint}>
                  Отдаёте эту вещь
                </small>
              </div>
            )}
            {current && (
              <div className={styles.details__receive}>
                <small className={styles.details__itemHint}>Вы получаете</small>
                <strong className={styles.details__itemTitle}>
                  {current.receives_item.title}
                </strong>
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
                <h2 className={styles.details__sectionTitle}>
                  Что нужно сделать
                </h2>
                <div className={styles.details__notice}>
                  Подтвердите участие. До подтверждения ваша вещь остаётся у вас.
                </div>
                <h3 className={styles.details__subsectionTitle}>
                  Подтверждения участников
                </h3>
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
                <h2 className={styles.details__sectionTitle}>
                  Хотите принять участие в этой цепочке?
                </h2>
                <h3 className={styles.details__subsectionTitle}>
                  Подтверждения участников
                </h3>
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

      {["confirmed", "delivering"].includes(exchange.status) && (
        current && user?.id && (
          <ExchangePickupStage
            currentUserId={user.id}
            exchangeStatus={exchange.status as "confirmed" | "delivering"}
            participants={exchange.participants}
          />
        )
      )}

      {exchange.status === "delivered" && (
        <>
          {isParticipant && (
            <section className={styles.details__receipt}>
              <div className={styles.details__receiptHeading}>
                <span className={styles.details__receiptStage}>
                  Этап получения
                </span>
                <h2 className={styles.details__sectionTitle}>
                  {receiptConfirmed
                    ? "Получение подтверждено"
                    : "Вы получили вещь?"}
                </h2>
                <p className={styles.details__receiptDescription}>
                  {receiptConfirmed
                    ? "Ожидаем подтверждения остальных участников."
                    : "Подтвердите получение только после проверки вещи."}
                </p>
                {current?.receives_item.pickup_point && (
                  <div className={styles.details__receiptPoint}>
                    <span className={styles.details__receiptPointLabel}>
                      Заберите вещь в ПВЗ
                    </span>
                    <strong className={styles.details__receiptPointName}>
                      {current.receives_item.pickup_point.name}
                    </strong>
                    <span className={styles.details__receiptPointAddress}>
                      {current.receives_item.pickup_point.address}
                    </span>
                  </div>
                )}
              </div>

              <div className={styles.details__participantProgress}>
                <h3 className={styles.details__subsectionTitle}>
                  Получение участников
                </h3>
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
          <h2 className={styles.details__resultTitle}>Обмен завершён</h2>
          <p className={styles.details__resultDescription}>
            Все участники получили свои вещи.
          </p>
          <ParticipantList
            currentUserId={user?.id}
            exchangeStatus={exchange.status}
            participants={exchange.participants}
          />
          {exchange.rating && (
            <ExchangeRating
              exchangeId={exchange.id}
              partnerNickname={ratedPartnerNickname}
              rating={exchange.rating}
            />
          )}
          <Link className={styles.details__resultAction} to="/exchanges">
            Вернуться к цепочкам
          </Link>
        </section>
      )}

      {exchange.status === "cancelled" && (
        <section className={styles.details__result}>
          <span
            className={`${styles.details__resultIcon} ${styles.details__resultIcon_error}`}
          >
            !
          </span>
          <h2 className={styles.details__resultTitle}>
            {cancelledByAdmin
              ? "Обмен отменён администратором"
              : "Один из участников отказался"}
          </h2>
          <p className={styles.details__resultDescription}>
            {cancelledByAdmin
              ? "Объявления освобождены, для них снова может начаться поиск обмена."
              : "Ваша вещь не передана и остаётся доступной."}
          </p>
          <div className={styles.details__resultButtons}>
            <Link className={styles.details__resultAction} to="/feed">
              Искать новую цепочку
            </Link>
            <Link
              className={`${styles.details__resultAction} ${styles.details__resultAction_secondary}`}
              to="/myItems"
            >
              Вернуть вещь в каталог
            </Link>
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
              ["confirmed", "delivering", "delivered"].includes(
                exchange.status,
              )
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
