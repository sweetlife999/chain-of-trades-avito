import {
  memo,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import { Link } from "react-router-dom";
import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import {
  getExchangeMessages,
  markExchangeMessagesRead,
  sendExchangeMessage,
} from "../../../../Api/exchanges/exchanges";
import type { TExchangeMessage } from "../../../../Api/exchanges/exchanges.types";
import { ReportMessageButton } from "../ReportMessageButton/ReportMessageButton";

type TProps = {
  exchangeId: string;
  currentUserId: string;
  readOnly?: boolean;
};

const formatTime = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));

const isSystemMessage = (message: TExchangeMessage) =>
  message.kind.toLowerCase() !== "text" || !message.author;

const getSystemMessageText = (message: TExchangeMessage) => {
  const nickname = message.author?.nickname;

  switch (message.kind.toLowerCase()) {
    case "participant_accepted":
      return nickname
        ? `${nickname} подтвердил участие в обмене`
        : "Участник подтвердил участие в обмене";
    case "participant_declined":
    case "participant_rejected":
      return nickname
        ? `${nickname} отказался от участия в обмене`
        : "Участник отказался от участия в обмене";
    case "participant_completed":
    case "participant_received":
      return nickname
        ? `${nickname} подтвердил получение вещи`
        : "Участник подтвердил получение вещи";
    case "participant_delivered_item":
      return nickname
        ? `${nickname} передал вещь в пункт выдачи`
        : "Участник передал вещь в пункт выдачи";
    case "exchange_confirmed":
      return "Все участники подтвердили обмен";
    case "exchange_delivering":
      return "Все вещи приняты — началась доставка";
    case "exchange_delivered":
      return "Вещи доставлены в пункты выдачи получателей";
    case "exchange_completed":
      return "Обмен успешно завершён";
    case "exchange_superseded":
      return "Обмен заменён новой цепочкой";
    default:
      return message.body ?? "Событие обмена";
  }
};

const ExchangeChatComponent = ({
  exchangeId,
  currentUserId,
  readOnly = false,
}: TProps) => {
  const [body, setBody] = useState("");
  const endRef = useRef<HTMLDivElement>(null);
  const lastMarkedMessageRef = useRef<string | null>(null);
  const queryClient = useQueryClient();
  const queryKey = ["exchanges", exchangeId, "messages"] as const;

  const messagesQuery = useQuery({
    queryKey,
    queryFn: () => getExchangeMessages(exchangeId),
    enabled: Boolean(exchangeId),
    refetchInterval: readOnly ? false : 3000,
    refetchIntervalInBackground: false,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    retry: false,
    staleTime: 0,
  });

  const sendMutation = useMutation({
    mutationFn: (messageBody: string) =>
      sendExchangeMessage(exchangeId, messageBody),
    onSuccess: (message) => {
      queryClient.setQueryData<TExchangeMessage[]>(queryKey, (messages = []) => {
        if (messages.some(({ id }) => id === message.id)) {
          return messages;
        }

        return [...messages, message];
      });
      queryClient.invalidateQueries({
        queryKey: ["exchanges"],
        exact: true,
      });
      setBody("");
    },
  });

  const messages = messagesQuery.data ?? [];
  const lastMessageId = messages.at(-1)?.id;

  useEffect(() => {
    endRef.current?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }, [messages.length]);

  useEffect(() => {
    if (!lastMessageId || lastMarkedMessageRef.current === lastMessageId) {
      return;
    }

    lastMarkedMessageRef.current = lastMessageId;

     markExchangeMessagesRead(exchangeId, lastMessageId)
      .then(() =>
        queryClient.invalidateQueries({
          queryKey: ["exchanges"],
          exact: true,
        }),
      )
      .catch(() => {
        if (lastMarkedMessageRef.current === lastMessageId) {
          lastMarkedMessageRef.current = null;
        }
      });
  }, [exchangeId, lastMessageId, queryClient]);

  const submitMessage = () => {
    const value = body.trim();

    if (readOnly || !value || sendMutation.isPending) {
      return;
    }

    sendMutation.mutate(value);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      submitMessage();
    }
  };

  return (
    <section className={styles.chat}>
      <div className={styles.chat__header}>
        <div className={styles.chat__heading}>
          <h2 className={styles.chat__title}>Чат цепочки</h2>
          <p className={styles.chat__description}>
            {readOnly
              ? "История обсуждения сохранена."
              : "Обсудите детали обмена с участниками."}
          </p>
        </div>
        <span className={styles.chat__mode}>
          {readOnly ? "Только чтение" : "Обновляется автоматически"}
        </span>
      </div>

      <div className={styles.chat__messages}>
        {messagesQuery.isPending && (
          <p className={styles.chat__state}>Загрузка сообщений...</p>
        )}

        {messagesQuery.isError && (
          <p className={styles.chat__state}>
            Не удалось загрузить историю сообщений.
          </p>
        )}

        {!messagesQuery.isPending &&
          !messagesQuery.isError &&
          messages.length === 0 && (
            <p className={styles.chat__state}>Сообщений пока нет.</p>
          )}

        {messages.map((message) => {
          const system = isSystemMessage(message);
          const own = message.author?.id === currentUserId;
          const authorProfilePath = message.author
            ? message.author.id === currentUserId
              ? "/profile"
              : `/profile/${message.author.id}`
            : "";

          if (system) {
            const systemText = getSystemMessageText(message);
            const authorPrefix = message.author
              ? `${message.author.nickname} `
              : "";
            const systemAction = authorPrefix && systemText.startsWith(authorPrefix)
              ? systemText.slice(authorPrefix.length)
              : systemText;

            return (
              <div className={styles.chat__system} key={message.id}>
                {message.author ? (
                  <>
                    <Link
                      className={styles.chat__systemAuthor}
                      to={authorProfilePath}
                    >
                      {message.author.nickname}
                    </Link>
                    <span className={styles.chat__systemText}>
                      {systemAction}
                    </span>
                  </>
                ) : (
                  <span className={styles.chat__systemText}>{systemText}</span>
                )}
                <time className={styles.chat__systemTime}>
                  {formatTime(message.created_at)}
                </time>
              </div>
            );
          }

          return (
            <article
              className={`${styles.chat__message} ${
                own ? styles.chat__message_own : ""
              }`}
              key={message.id}
            >
              {!own && message.author && (
                <Link
                  aria-label={`Профиль ${message.author.nickname}`}
                  className={styles.chat__avatarLink}
                  to={authorProfilePath}
                >
                  <span className={styles.chat__avatar}>
                    {message.author.photo_url ? (
                      <img
                        className={styles.chat__avatarImage}
                        alt={message.author.nickname}
                        src={message.author.photo_url}
                      />
                    ) : (
                      message.author.nickname.charAt(0).toUpperCase()
                    )}
                  </span>
                </Link>
              )}
              <div
                className={`${styles.chat__bubble} ${
                  !own ? styles.chat__bubble_reportable : ""
                }`}
              >
                {!own && message.author && (
                  <Link
                    className={styles.chat__authorLink}
                    to={authorProfilePath}
                  >
                    <strong className={styles.chat__authorName}>
                      {message.author.nickname}
                    </strong>
                  </Link>
                )}
                <p className={styles.chat__messageText}>
                  {message.body ?? ""}
                </p>
                <time className={styles.chat__messageTime}>
                  {formatTime(message.created_at)}
                </time>
                {!own && (
                  <ReportMessageButton
                    className={styles.chat__reportAction}
                    messageId={message.id}
                  />
                )}
              </div>
            </article>
          );
        })}

        <div className={styles.chat__end} ref={endRef} />
      </div>

      {readOnly ? (
        <div className={styles.chat__readOnly}>
          Обмен закрыт. Новые сообщения отправлять нельзя.
        </div>
      ) : (
        <form
          className={styles.chat__form}
          onSubmit={(event) => {
            event.preventDefault();
            submitMessage();
          }}
        >
          <textarea
            className={styles.chat__textarea}
            maxLength={2000}
            onChange={(event) => setBody(event.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Напишите сообщение..."
            rows={2}
            value={body}
          />
          <div className={styles.chat__formFooter}>
            <span className={styles.chat__counter}>{body.length} / 2000</span>
            <button
              className={styles.chat__submit}
              disabled={!body.trim() || sendMutation.isPending}
              type="submit"
            >
              {sendMutation.isPending ? "Отправляем..." : "Отправить"}
            </button>
          </div>
          {sendMutation.isError && (
            <p className={styles.chat__error}>
              Не удалось отправить сообщение.
            </p>
          )}
        </form>
      )}
    </section>
  );
};

export const ExchangeChat = memo(ExchangeChatComponent);
