import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Bot, CheckCircle2, MessageSquareText, ShieldX, X } from "lucide-react";

import {
  confirmAdminAntiscamCase,
  dismissAdminAntiscamCase,
  getAdminAntiscamCases,
  getAdminAntiscamMessages,
  getAdminErrorMessage,
} from "../../../../Api/admin/admin";
import type {
  TAdminAntiscamCategory,
  TAdminAntiscamStatus,
} from "../../../../Api/admin/admin.types";
import { Button } from "../../../UI/Button/Button";
import { Loader } from "../../../UI/Loader/Loader";
import styles from "./Styles.module.scss";

type TStatus = "all" | TAdminAntiscamStatus;
type TCategory = "all" | TAdminAntiscamCategory;

const categoryLabels: Record<TAdminAntiscamCategory, string> = {
  credentials: "Секретные данные",
  external_payment: "Оплата вне сервиса",
  external_contact: "Увод из сервиса",
  phishing: "Подозрительная ссылка",
  pressure: "Давление и срочность",
  other: "Другое",
};

const statusLabels = {
  open: "Открыто",
  resolved: "Нарушение",
  dismissed: "Ложное срабатывание",
} as const;

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));

export const AdminAntiscam = () => {
  const [status, setStatus] = useState<TStatus>("open");
  const [category, setCategory] = useState<TCategory>("all");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [comment, setComment] = useState("");
  const queryClient = useQueryClient();
  const casesQuery = useQuery({
    queryKey: ["admin", "antiscam", { status, category }],
    queryFn: () => getAdminAntiscamCases({
      status: status === "all" ? undefined : status,
      category: category === "all" ? undefined : category,
      limit: 50,
    }),
    refetchInterval: 15_000,
    retry: false,
  });
  const selected = casesQuery.data?.cases.find(({ id }) => id === selectedId);
  const messagesQuery = useQuery({
    queryKey: ["admin", "antiscam", selectedId, "messages"],
    queryFn: () => getAdminAntiscamMessages(selectedId!),
    enabled: Boolean(selectedId),
    retry: false,
  });
  const decideMutation = useMutation({
    mutationFn: (action: "confirm" | "dismiss") =>
      action === "confirm"
        ? confirmAdminAntiscamCase(selectedId!, comment.trim())
        : dismissAdminAntiscamCase(selectedId!, comment.trim()),
    onSuccess: async () => {
      setComment("");
      setSelectedId(null);
      await queryClient.invalidateQueries({ queryKey: ["admin", "antiscam"] });
    },
  });
  const evidenceIDs = new Set(messagesQuery.data?.evidence_message_ids ?? []);

  return (
    <section className={styles.antiscam} aria-labelledby="admin-antiscam-title">
      <header className={styles.antiscam__header}>
        <div className={styles.antiscam__heading}>
          <Bot aria-hidden="true" />
          <div>
            <h2 id="admin-antiscam-title">AI-антискам</h2>
            <p>Модель и правила отмечают риск, окончательное решение принимает администратор.</p>
          </div>
        </div>
        <strong>Найдено: {casesQuery.data?.pagination.total ?? 0}</strong>
      </header>

      <div className={styles.antiscam__filters}>
        <label>Статус<select value={status} onChange={(event) => setStatus(event.target.value as TStatus)}><option value="all">Все</option><option value="open">Открытые</option><option value="resolved">Подтверждённые</option><option value="dismissed">Ложные</option></select></label>
        <label>Категория<select value={category} onChange={(event) => setCategory(event.target.value as TCategory)}><option value="all">Все</option>{Object.entries(categoryLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
      </div>

      {casesQuery.isPending && <Loader text="Загружаем AI-подозрения..." />}
      {casesQuery.isError && <p className={styles.antiscam__state} role="alert"><AlertTriangle /> Не удалось загрузить очередь антискама.</p>}
      {!casesQuery.isPending && (casesQuery.data?.cases.length ?? 0) === 0 && <p className={styles.antiscam__state}>Подозрений по выбранным фильтрам нет.</p>}
      <div className={styles.antiscam__list}>
        {casesQuery.data?.cases.map((item) => (
          <button key={item.id} type="button" className={styles.antiscam__card} onClick={() => { setSelectedId(item.id); setComment(""); }}>
            <div className={styles.antiscam__cardTop}><span className={styles[`antiscam__risk_${item.risk >= 80 ? "high" : "medium"}`]}>Риск {item.risk}%</span><span>{statusLabels[item.status]}</span><time>{formatDate(item.updated_at)}</time></div>
            <strong>{categoryLabels[item.category]}</strong>
            <blockquote>{item.latest_evidence.body}</blockquote>
            <span>{item.suspect.nickname} · сигналов: {item.evidence_count}</span>
          </button>
        ))}
      </div>

      {selected && (
        <div className={styles.antiscam__overlay} onMouseDown={(event) => {
          if (event.target === event.currentTarget && !decideMutation.isPending) {
            setSelectedId(null);
          }
        }}>
          <section className={styles.antiscam__details} role="dialog" aria-modal="true" aria-labelledby="antiscam-case-title">
            <header><div><span>AI-подозрение · риск {selected.risk}%</span><h3 id="antiscam-case-title">{categoryLabels[selected.category]}</h3></div><button type="button" aria-label="Закрыть" onClick={() => setSelectedId(null)}><X /></button></header>
            <p className={styles.antiscam__reason}>{selected.reason}</p>
            <p>Пользователь: <strong>{selected.suspect.nickname}</strong></p>
            <a href={`/exchanges/${selected.exchange_id}`}>Открыть обмен</a>
            <div className={styles.antiscam__conversation}>
              <h4><MessageSquareText /> Переписка</h4>
              {messagesQuery.isPending && <Loader text="Загружаем переписку..." />}
              {messagesQuery.data?.messages.map((message) => (
                <article key={message.id} className={evidenceIDs.has(message.id) ? styles.antiscam__evidence : ""}>
                  <div><strong>{message.author?.nickname ?? "Событие обмена"}</strong>{evidenceIDs.has(message.id) && <span>AI-сигнал</span>}<time>{formatDate(message.created_at)}</time></div>
                  <p>{message.body ?? message.kind}</p>
                </article>
              ))}
            </div>
            {selected.status === "open" && <div className={styles.antiscam__decision}>
              <label>Комментарий администратора<textarea value={comment} onChange={(event) => setComment(event.target.value)} rows={3} placeholder="Почему это нарушение или ложное срабатывание" /></label>
              {decideMutation.isError && <p role="alert">{getAdminErrorMessage(decideMutation.error, "Не удалось сохранить решение")}</p>}
              <div><Button color="danger" disabled={!comment.trim() || decideMutation.isPending} onClick={() => decideMutation.mutate("confirm")}><CheckCircle2 /> Подтвердить нарушение</Button><Button color="light" disabled={!comment.trim() || decideMutation.isPending} onClick={() => decideMutation.mutate("dismiss")}><ShieldX /> Ложное срабатывание</Button></div>
            </div>}
          </section>
        </div>
      )}
    </section>
  );
};
