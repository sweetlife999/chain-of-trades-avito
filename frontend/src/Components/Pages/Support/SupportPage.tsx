import {
  type FormEvent,
  type KeyboardEvent,
  memo,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import clsx from "clsx";
import { CheckCircle2, Headset, Plus, Send, X } from "lucide-react";
import { useSearchParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  closeSupportThread,
  createSupportThread,
  getSupportError,
  getSupportMessages,
  getSupportThreads,
  sendSupportMessage,
} from "../../../Api/support/support";
import type { TSupportThread } from "../../../Api/support/support.types";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { AuthRequiredState } from "../../UI/AuthRequiredState/AuthRequiredState";

const statusLabels: Record<TSupportThread["status"], string> = {
  open: "Ждёт ответа",
  in_progress: "В работе",
  closed: "Закрыто",
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));

const SupportPageComponent = () => {
  const { isAuth, user } = useAuthSelector();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [creating, setCreating] = useState(false);
  const [subject, setSubject] = useState("");
  const [firstMessage, setFirstMessage] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const messageInputRef = useRef<HTMLTextAreaElement>(null);

  const restoreMessageFocus = () => {
    window.requestAnimationFrame(() => messageInputRef.current?.focus());
  };

  const selectedID = searchParams.get("thread") ?? "";
  const threadsQuery = useQuery({
    queryKey: ["support", "threads"],
    queryFn: getSupportThreads,
    enabled: isAuth,
    refetchInterval: 5000,
  });
  const threads = useMemo(() => threadsQuery.data ?? [], [threadsQuery.data]);
  const activeThread = threads.find((thread) => thread.status !== "closed");

  useEffect(() => {
    if (selectedID || threads.length === 0) {
      return;
    }
    setSearchParams({ thread: (activeThread ?? threads[0]).id }, { replace: true });
  }, [activeThread, selectedID, setSearchParams, threads]);

  const messagesQuery = useQuery({
    queryKey: ["support", "messages", selectedID],
    queryFn: () => getSupportMessages(selectedID),
    enabled: isAuth && Boolean(selectedID),
    refetchInterval: 3000,
  });

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["support"] }),
      queryClient.invalidateQueries({ queryKey: ["notifications"] }),
    ]);
  };

  const createMutation = useMutation({
    mutationFn: createSupportThread,
    onSuccess: async (thread) => {
      setCreating(false);
      setSubject("");
      setFirstMessage("");
      setError("");
      setSearchParams({ thread: thread.id });
      await refresh();
    },
    onError: (requestError) => {
      setError(getSupportError(requestError, "Не удалось создать обращение"));
    },
  });
  const sendMutation = useMutation({
    mutationFn: sendSupportMessage,
    onSuccess: async () => {
      setMessage("");
      setError("");
      restoreMessageFocus();
      await refresh();
      restoreMessageFocus();
    },
    onError: (requestError) => {
      setError(getSupportError(requestError, "Не удалось отправить сообщение"));
      restoreMessageFocus();
    },
  });
  const closeMutation = useMutation({
    mutationFn: closeSupportThread,
    onSuccess: refresh,
    onError: (requestError) =>
      setError(getSupportError(requestError, "Не удалось закрыть обращение")),
  });

  if (!isAuth) {
    return (
      <main className={styles.support}>
        <AuthRequiredState
          description="Войдите, чтобы написать в поддержку и увидеть историю обращений."
          returnTo="/support"
          title="Поддержка доступна после входа"
        />
      </main>
    );
  }

  const selectedThread = messagesQuery.data?.thread;
  const canCreate = !activeThread;

  const submitCreate = (event: FormEvent) => {
    event.preventDefault();
    createMutation.mutate({ subject: subject.trim(), body: firstMessage.trim() });
  };
  const sendCurrentMessage = () => {
    const value = message.trim();

    if (!selectedID || !value || sendMutation.isPending) {
      return;
    }

    sendMutation.mutate({ threadID: selectedID, body: value });
  };
  const submitMessage = (event: FormEvent) => {
    event.preventDefault();
    sendCurrentMessage();
  };
  const handleMessageKeyDown = (
    event: KeyboardEvent<HTMLTextAreaElement>,
  ) => {
    if (
      event.key === "Enter" &&
      !event.shiftKey &&
      !event.nativeEvent.isComposing
    ) {
      event.preventDefault();
      sendCurrentMessage();
    }
  };

  return (
    <main className={styles.support}>
      <header className={styles.support__header}>
        <div>
          <h1 className={styles.support__title}>Поддержка</h1>
          <p className={styles.support__subtitle}>Задайте вопрос — ответ придёт прямо в этот чат.</p>
        </div>
        <button
          className={styles.support__newButton}
          disabled={!canCreate}
          title={!canCreate ? "Сначала закройте текущее обращение" : undefined}
          type="button"
          onClick={() => setCreating(true)}
        >
          <Plus size={18} /> Новое обращение
        </button>
      </header>

      {creating && (
        <form className={styles.support__create} onSubmit={submitCreate}>
          <button aria-label="Закрыть форму" className={styles.support__closeForm} type="button" onClick={() => setCreating(false)}>
            <X size={20} />
          </button>
          <h2>Новое обращение</h2>
          <label>Тема<input maxLength={160} required value={subject} onChange={(event) => setSubject(event.target.value)} /></label>
          <label>Что случилось?<textarea maxLength={4000} required rows={5} value={firstMessage} onChange={(event) => setFirstMessage(event.target.value)} /></label>
          {error && <p className={styles.support__error}>{error}</p>}
          <button className={styles.support__primary} disabled={createMutation.isPending} type="submit">
            {createMutation.isPending ? "Отправляем..." : "Отправить обращение"}
          </button>
        </form>
      )}

      <div className={styles.support__workspace}>
        <aside className={styles.support__sidebar}>
          <h2>Мои обращения</h2>
          {threadsQuery.isPending && <p className={styles.support__state}>Загрузка...</p>}
          {threadsQuery.isError && <p className={styles.support__error}>Не удалось загрузить обращения.</p>}
          {!threadsQuery.isPending && threads.length === 0 && (
            <div className={styles.support__emptySmall}><Headset size={28} /><span>Обращений пока нет</span></div>
          )}
          {threads.map((thread) => (
            <button
              className={clsx(styles.support__thread, thread.id === selectedID && styles.support__thread_active)}
              key={thread.id}
              type="button"
              onClick={() => setSearchParams({ thread: thread.id })}
            >
              <span className={styles.support__threadTop}><strong>{thread.subject}</strong>{thread.unread_count > 0 && <b>{thread.unread_count}</b>}</span>
              <span>{statusLabels[thread.status]}</span>
              <small>{formatDate(thread.last_message_at ?? thread.created_at)}</small>
            </button>
          ))}
        </aside>

        <section className={styles.support__chat}>
          {!selectedID && (
            <div className={styles.support__empty}><Headset size={32} /><h2>Напишите нам</h2><p>Создайте обращение, и команда поддержки поможет разобраться.</p></div>
          )}
          {selectedID && messagesQuery.isPending && <p className={styles.support__state}>Загружаем переписку...</p>}
          {selectedID && messagesQuery.isError && <p className={styles.support__error}>Не удалось открыть переписку.</p>}
          {selectedThread && (
            <>
              <div className={styles.support__chatHeader}>
                <div><h2>{selectedThread.subject}</h2><span>{statusLabels[selectedThread.status]}{selectedThread.assigned_admin ? ` · ${selectedThread.assigned_admin.nickname}` : ""}</span></div>
                {selectedThread.status !== "closed" && (
                  <button disabled={closeMutation.isPending} type="button" onClick={() => closeMutation.mutate(selectedThread.id)}>
                    <CheckCircle2 size={17} /> Закрыть
                  </button>
                )}
              </div>
              <div className={styles.support__messages}>
                {(messagesQuery.data?.messages ?? []).map((item) => {
                  const own = item.author.id === user?.id;
                  return (
                    <article className={clsx(styles.support__message, own && styles.support__message_own)} key={item.id}>
                      <strong>{own ? "Вы" : item.author.nickname}</strong>
                      <p>{item.body}</p><time>{formatDate(item.created_at)}</time>
                    </article>
                  );
                })}
              </div>
              {error && <p className={styles.support__error}>{error}</p>}
              {selectedThread.status !== "closed" ? (
                <form className={styles.support__composer} onSubmit={submitMessage}>
                  <textarea
                    aria-label="Сообщение"
                    maxLength={4000}
                    placeholder="Напишите сообщение..."
                    ref={messageInputRef}
                    required
                    rows={2}
                    value={message}
                    onChange={(event) => setMessage(event.target.value)}
                    onKeyDown={handleMessageKeyDown}
                  />
                  <button aria-label="Отправить" disabled={sendMutation.isPending || !message.trim()} type="submit"><Send size={20} /></button>
                </form>
              ) : (
                <p className={styles.support__closed}>Обращение закрыто. Можно создать новое.</p>
              )}
            </>
          )}
        </section>
      </div>
    </main>
  );
};

export const SupportPage = memo(SupportPageComponent);
