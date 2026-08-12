import {
  memo,
  useEffect,
  useRef,
  useMemo,
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
import { Mascot } from "../../../UI/Mascot/Mascot";
import { useMascot } from "../../../../Hooks/useMascot";
import type { MascotEvent } from "../../../../Features/Mascot/mascot.types";

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


const starterHints = [
  "Привет! Давайте договоримся, как удобнее передать вещи?",
  "Когда вам удобно передать вещь?",
  "В каком пункте выдачи вам удобнее встретиться?",
];

const activeHints = [
  "Когда вам удобно передать вещь?",
  "В каком пункте выдачи вам удобнее?",
  "Всё в силе?",
  "Напишите, пожалуйста, когда передадите вещь.",
];

const getMascotEventForSystemMessage = (message: TExchangeMessage): MascotEvent | null => {
  switch (message.kind.toLowerCase()) {
    case "participant_accepted":
      return "PARTICIPANT_ACCEPTED";
    case "participant_declined":
    case "participant_rejected":
      return "PARTICIPANT_DECLINED";
    case "participant_delivered_item":
      return "PARTICIPANT_DELIVERED";
    case "exchange_confirmed":
      return "EXCHANGE_CONFIRMED";
    case "exchange_delivering":
      return "EXCHANGE_DELIVERING";
    case "exchange_delivered":
      return "EXCHANGE_DELIVERED";
    case "exchange_completed":
      return "EXCHANGE_COMPLETED";
    default:
      return null;
  }
};

const ExchangeChatComponent = ({
  exchangeId,
  currentUserId,
  readOnly = false,
}: TProps) => {
  const [body, setBody] = useState("");
  const { reactTo } = useMascot();
  const endRef = useRef<HTMLDivElement>(null);
  const lastMarkedMessageRef = useRef<string | null>(null);
  const lastObservedMessageRef = useRef<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
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
      reactTo("MESSAGE_SENT_SUCCESS");
    },
    onError: () => reactTo("ERROR"),
  });

  const messages = useMemo(() => messagesQuery.data ?? [], [messagesQuery.data]);
  const lastMessageId = messages.at(-1)?.id;
  const hints = messages.length === 0 ? starterHints : activeHints;

  useEffect(() => {
    reactTo("CHAT_OPENED");

    const hintTimer = window.setTimeout(() => reactTo("HINT_SHOWN"), 1200);
    return () => window.clearTimeout(hintTimer);
  }, [exchangeId, reactTo]);

  useEffect(() => {
    endRef.current?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  }, [messages.length]);

  useEffect(() => {
    const latest = messages.at(-1);

    if (!latest) {
      return;
    }

    if (lastObservedMessageRef.current === null) {
      lastObservedMessageRef.current = latest.id;
      return;
    }

    if (lastObservedMessageRef.current === latest.id) {
      return;
    }

    lastObservedMessageRef.current = latest.id;

    if (isSystemMessage(latest)) {
      const event = getMascotEventForSystemMessage(latest);
      if (event) {
        reactTo(event);
      }
      return;
    }

    if (latest.author?.id !== currentUserId) {
      reactTo("MESSAGE_RECEIVED");
    }
  }, [currentUserId, messages, reactTo]);

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

    reactTo("MESSAGE_SENT");
    sendMutation.mutate(value);
  };

  const handleHintSelect = (hint: string) => {
    setBody(hint);
    reactTo("HINT_SELECTED");
    window.requestAnimationFrame(() => {
      textareaRef.current?.focus();
      textareaRef.current?.setSelectionRange(hint.length, hint.length);
    });
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
          <div className={styles.chat__assistant}>
            <Mascot size="small" placement="chat" />
            <div className={styles.chat__hints}>
              <span className={styles.chat__hintsLabel}>Быстрые подсказки</span>
              <div className={styles.chat__hintList}>
                {hints.map((hint) => (
                  <button
                    className={styles.chat__hint}
                    key={hint}
                    type="button"
                    onClick={() => handleHintSelect(hint)}
                    onMouseEnter={() => reactTo("HINT_SHOWN")}
                    onFocus={() => reactTo("HINT_SHOWN")}
                  >
                    {hint}
                  </button>
                ))}
              </div>
            </div>
          </div>
          <textarea
            ref={textareaRef}
            className={styles.chat__textarea}
            maxLength={2000}
            onChange={(event) => {
              const nextBody = event.target.value;
              if (!body && nextBody) {
                reactTo("USER_TYPING");
              }
              setBody(nextBody);
            }}
            onFocus={() => body && reactTo("USER_TYPING")}
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
