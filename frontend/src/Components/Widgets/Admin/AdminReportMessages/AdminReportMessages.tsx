import { useQuery } from "@tanstack/react-query";
import { MessageSquareText } from "lucide-react";

import { getAdminReportMessages } from "../../../../Api/admin/admin";
import type { TExchangeMessage } from "../../../../Api/exchanges/exchanges.types";
import { Loader } from "../../../UI/Loader/Loader";
import styles from "./Styles.module.scss";

type TProps = {
  reportId: string;
  reportedMessageId: string;
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "short",
  }).format(new Date(value));

const getSystemMessage = (message: TExchangeMessage) => {
  const nickname = message.author?.nickname ?? "Участник";

  switch (message.kind) {
    case "participant_accepted":
      return nickname + " подтвердил участие";
    case "participant_declined":
      return nickname + " отказался от обмена";
    case "participant_completed":
      return nickname + " подтвердил получение";
    case "participant_delivered_item":
      return nickname + " передал вещь в ПВЗ";
    case "exchange_confirmed":
      return "Все участники подтвердили обмен";
    case "exchange_delivering":
      return "Началась доставка вещей";
    case "exchange_delivered":
      return "Вещи доставлены получателям";
    case "exchange_completed":
      return "Обмен завершён";
    case "exchange_superseded":
      return "Обмен заменён новой цепочкой";
    default:
      return message.body ?? "Событие обмена";
  }
};

export const AdminReportMessages = ({
  reportId,
  reportedMessageId,
}: TProps) => {
  const messagesQuery = useQuery({
    queryKey: ["admin", "reports", reportId, "messages"],
    queryFn: () => getAdminReportMessages(reportId),
    retry: false,
  });

  const messages = messagesQuery.data?.messages ?? [];

  return (
    <section className={styles.reportMessages}>
      <div className={styles.reportMessages__header}>
        <MessageSquareText
          aria-hidden="true"
          className={styles.reportMessages__headerIcon}
        />
        <div className={styles.reportMessages__heading}>
          <h3 className={styles.reportMessages__title}>Переписка по жалобе</h3>
          <p className={styles.reportMessages__description}>
            Режим просмотра: отправка сообщений отключена.
          </p>
        </div>
      </div>

      {messagesQuery.isPending && (
        <div className={styles.reportMessages__state}>
          <Loader text="Загружаем переписку..." />
        </div>
      )}

      {messagesQuery.isError && (
        <p className={styles.reportMessages__state} role="alert">
          Не удалось загрузить переписку.
        </p>
      )}

      {!messagesQuery.isPending &&
        !messagesQuery.isError &&
        messages.length === 0 && (
          <p className={styles.reportMessages__state}>
            В переписке пока нет сообщений.
          </p>
        )}

      {messages.length > 0 && (
        <div className={styles.reportMessages__list}>
          {messages.map((message) => {
            const isText = message.kind === "text" && message.author;
            const reported = message.id === reportedMessageId;

            if (!isText) {
              return (
                <div className={styles.reportMessages__system} key={message.id}>
                  <span className={styles.reportMessages__systemText}>
                    {getSystemMessage(message)}
                  </span>
                  <time className={styles.reportMessages__time}>
                    {formatDate(message.created_at)}
                  </time>
                </div>
              );
            }

            return (
              <article
                className={[
                  styles.reportMessages__message,
                  reported ? styles.reportMessages__message_reported : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
                key={message.id}
              >
                <div className={styles.reportMessages__messageHeader}>
                  <strong className={styles.reportMessages__author}>
                    {message.author?.nickname}
                  </strong>
                  {reported && (
                    <span className={styles.reportMessages__reportedLabel}>
                      Сообщение из жалобы
                    </span>
                  )}
                  <time className={styles.reportMessages__time}>
                    {formatDate(message.created_at)}
                  </time>
                </div>
                <p className={styles.reportMessages__body}>{message.body}</p>
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
};
