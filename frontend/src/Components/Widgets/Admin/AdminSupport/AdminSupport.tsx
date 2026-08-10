import { type FormEvent, memo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import clsx from "clsx";
import { CheckCircle2, Headset, Send, UserCheck } from "lucide-react";
import { useSearchParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  assignAdminSupportThread,
  closeAdminSupportThread,
  getAdminSupportMessages,
  getAdminSupportThreads,
  getSupportError,
  sendAdminSupportMessage,
} from "../../../../Api/support/support";
import type { TSupportThread } from "../../../../Api/support/support.types";
import { useAuthSelector } from "../../../../Hooks/useAuthDispatch";

type TFilter = "" | TSupportThread["status"];

const filters: Array<{ value: TFilter; label: string }> = [
  { value: "", label: "Все" },
  { value: "open", label: "Новые" },
  { value: "in_progress", label: "В работе" },
  { value: "closed", label: "Закрытые" },
];

const statusLabels: Record<TSupportThread["status"], string> = {
  open: "Новое",
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

const AdminSupportComponent = () => {
  const { user } = useAuthSelector();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedID = searchParams.get("thread") ?? "";
  const [filter, setFilter] = useState<TFilter>("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const listQuery = useQuery({
    queryKey: ["admin-support", "threads", filter],
    queryFn: () => getAdminSupportThreads({ status: filter || undefined, limit: 100 }),
    refetchInterval: 4000,
  });
  const messagesQuery = useQuery({
    queryKey: ["admin-support", "messages", selectedID],
    queryFn: () => getAdminSupportMessages(selectedID),
    enabled: Boolean(selectedID),
    refetchInterval: 3000,
  });

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["admin-support"] }),
      queryClient.invalidateQueries({ queryKey: ["notifications"] }),
    ]);
  };
  const mutationError = (requestError: unknown, fallback: string) =>
    setError(getSupportError(requestError, fallback));
  const assignMutation = useMutation({
    mutationFn: assignAdminSupportThread,
    onSuccess: refresh,
    onError: (requestError) => mutationError(requestError, "Не удалось взять обращение"),
  });
  const sendMutation = useMutation({
    mutationFn: sendAdminSupportMessage,
    onSuccess: async () => {
      setMessage("");
      setError("");
      await refresh();
    },
    onError: (requestError) => mutationError(requestError, "Не удалось отправить ответ"),
  });
  const closeMutation = useMutation({
    mutationFn: closeAdminSupportThread,
    onSuccess: refresh,
    onError: (requestError) => mutationError(requestError, "Не удалось закрыть обращение"),
  });

  const thread = messagesQuery.data?.thread;
  const assignedToMe = thread?.assigned_admin?.id === user?.id;
  const openThread = (id: string) => {
    const next = new URLSearchParams(searchParams);
    next.set("section", "support");
    next.set("thread", id);
    setSearchParams(next);
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (selectedID && message.trim()) {
      sendMutation.mutate({ threadID: selectedID, body: message.trim() });
    }
  };

  return (
    <div className={styles.support}>
      <div className={styles.support__top}>
        <div><h2>Обращения в поддержку</h2><p>Очередь вопросов пользователей, не связанная с конкретной сделкой.</p></div>
        <div className={styles.support__filters}>
          {filters.map((item) => (
            <button className={filter === item.value ? styles.support__filter_active : ""} key={item.value} type="button" onClick={() => setFilter(item.value)}>{item.label}</button>
          ))}
        </div>
      </div>

      <div className={styles.support__workspace}>
        <aside className={styles.support__queue}>
          {listQuery.isPending && <p>Загрузка...</p>}
          {listQuery.isError && <p className={styles.support__error}>Не удалось загрузить очередь.</p>}
          {listQuery.data?.threads.length === 0 && <div className={styles.support__empty}><Headset size={28} />Обращений нет</div>}
          {listQuery.data?.threads.map((item) => (
            <button className={clsx(styles.support__thread, selectedID === item.id && styles.support__thread_active)} key={item.id} type="button" onClick={() => openThread(item.id)}>
              <span><strong>{item.subject}</strong>{item.unread_count > 0 && <b>{item.unread_count}</b>}</span>
              <span>{item.user?.nickname ?? "Пользователь"} · {statusLabels[item.status]}</span>
              <small>{formatDate(item.last_message_at ?? item.created_at)}</small>
            </button>
          ))}
        </aside>

        <section className={styles.support__chat}>
          {!selectedID && <div className={styles.support__empty}><Headset size={44} /><strong>Выберите обращение</strong></div>}
          {selectedID && messagesQuery.isPending && <p className={styles.support__state}>Загрузка переписки...</p>}
          {selectedID && messagesQuery.isError && <p className={styles.support__error}>Не удалось открыть обращение.</p>}
          {thread && (
            <>
              <header className={styles.support__chatHeader}>
                <div><h3>{thread.subject}</h3><span>{thread.user?.nickname} · {statusLabels[thread.status]}{thread.assigned_admin ? ` · админ ${thread.assigned_admin.nickname}` : ""}</span></div>
                <div className={styles.support__actions}>
                  {thread.status === "open" && <button disabled={assignMutation.isPending} type="button" onClick={() => assignMutation.mutate(thread.id)}><UserCheck size={17} />Взять в работу</button>}
                  {thread.status === "in_progress" && assignedToMe && <button disabled={closeMutation.isPending} type="button" onClick={() => closeMutation.mutate(thread.id)}><CheckCircle2 size={17} />Закрыть</button>}
                </div>
              </header>
              <div className={styles.support__messages}>
                {(messagesQuery.data?.messages ?? []).map((item) => {
                  const adminMessage = item.author.is_admin;
                  return <article className={clsx(styles.support__message, adminMessage && styles.support__message_admin)} key={item.id}><strong>{item.author.nickname}{adminMessage ? " · администратор" : ""}</strong><p>{item.body}</p><time>{formatDate(item.created_at)}</time></article>;
                })}
              </div>
              {error && <p className={styles.support__error}>{error}</p>}
              {thread.status === "in_progress" && assignedToMe ? (
                <form className={styles.support__composer} onSubmit={submit}>
                  <textarea maxLength={4000} placeholder="Ответ пользователю..." required rows={2} value={message} onChange={(event) => setMessage(event.target.value)} />
                  <button aria-label="Отправить" disabled={sendMutation.isPending || !message.trim()} type="submit"><Send size={20} /></button>
                </form>
              ) : thread.status !== "closed" ? (
                <p className={styles.support__hint}>{thread.assigned_admin ? "Обращение ведёт другой администратор." : "Возьмите обращение в работу, чтобы ответить."}</p>
              ) : <p className={styles.support__hint}>Обращение закрыто.</p>}
            </>
          )}
        </section>
      </div>
    </div>
  );
};

export const AdminSupport = memo(AdminSupportComponent);
