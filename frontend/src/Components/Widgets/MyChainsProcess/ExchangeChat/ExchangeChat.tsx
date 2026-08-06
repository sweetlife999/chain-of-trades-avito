import {
  memo,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import {
  getExchangeMessages,
  sendExchangeMessage,
} from "../../../../Api/exchanges/exchanges";
import type { TExchangeMessage } from "../../../../Api/exchanges/exchanges.types";

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

const isSystemMessage = (message: TExchangeMessage) => {
  const kind = message.kind.toLowerCase();

  return !message.author || kind.includes("system") || kind.includes("event");
};

const ExchangeChatComponent = ({
  exchangeId,
  currentUserId,
  readOnly = false,
}: TProps) => {
  const [body, setBody] = useState("");
  const endRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();
  const queryKey = ["exchanges", exchangeId, "messages"] as const;

  const messagesQuery = useQuery({
    queryKey,
    queryFn: () => getExchangeMessages(exchangeId),
    enabled: Boolean(exchangeId),
    refetchInterval: readOnly ? false : 3000,
    refetchIntervalInBackground: false,
    retry: false,
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

  // useEffect(() => {
  //   endRef.current?.scrollIntoView({
  //     behavior: "smooth",
  //     block: "end",
  //   });
  // }, [messages.length]);

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
      <header className={styles.chat__header}>
        <div>
          <h2>Чат цепочки</h2>
          <p>
            {readOnly
              ? "История обсуждения сохранена."
              : "Обсудите детали обмена с участниками."}
          </p>
        </div>
        <span>
          {readOnly ? "Только чтение" : "Обновляется автоматически"}
        </span>
      </header>

      <div className={styles.chat__messages}>
        {messagesQuery.isPending && (
          <p className={styles.chat__state}>Загрузка сообщений...</p>
        )}

        {messagesQuery.isError && (
          <p className={styles.chat__state}>
            Не удалось загрузить сообщения.
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

          if (system) {
            return (
              <div className={styles.chat__system} key={message.id}>
                <span>{message.body}</span>
                <time>{formatTime(message.created_at)}</time>
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
              {!own && (
                <span className={styles.chat__avatar}>
                  {message.author?.photo_url ? (
                    <img
                      alt={message.author.nickname}
                      src={message.author.photo_url}
                    />
                  ) : (
                    message.author?.nickname.charAt(0).toUpperCase()
                  )}
                </span>
              )}
              <div>
                {!own && <strong>{message.author?.nickname}</strong>}
                <p>{message.body}</p>
                <time>{formatTime(message.created_at)}</time>
              </div>
            </article>
          );
        })}

        <div ref={endRef} />
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
            maxLength={2000}
            onChange={(event) => setBody(event.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Напишите сообщение..."
            rows={2}
            value={body}
          />
          <div className={styles.chat__formFooter}>
            <span>{body.length} / 2000</span>
            <button
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