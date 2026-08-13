import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Sparkles } from "lucide-react";

import {
  createItemAISuggestion,
  getItemAISuggestion,
  getItemErrorMessage,
} from "../../../Api/items/items";
import type { TItemAISuggestion } from "../../../Api/items/items.types";
import { Button } from "../Button/Button";
import styles from "./Styles.module.scss";

type TProps = {
  onApply: (suggestion: TItemAISuggestion) => void;
  disabled?: boolean;
};

export const ItemAIAssistant = ({ onApply, disabled = false }: TProps) => {
  const [input, setInput] = useState("");
  const [jobId, setJobId] = useState<string>();
  const [appliedJobId, setAppliedJobId] = useState<string>();

  const submitMutation = useMutation({
    mutationFn: createItemAISuggestion,
    onSuccess: (job) => {
      setJobId(job.id);
      setAppliedJobId(undefined);
    },
  });

  const jobQuery = useQuery({
    queryKey: ["item-ai-suggestion", jobId],
    queryFn: () => getItemAISuggestion(jobId ?? ""),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === "completed" || status === "failed" ? false : 1200;
    },
  });

  const job = jobQuery.data;
  const suggestion =
    job?.status === "completed" && job.id !== appliedJobId
      ? job.suggestion
      : undefined;
  const message = submitMutation.isError
    ? getItemErrorMessage(
        submitMutation.error,
        "Не удалось запустить ИИ-помощника",
      )
    : jobQuery.isError
      ? getItemErrorMessage(jobQuery.error, "Не удалось получить подсказку")
      : job?.status === "failed"
        ? (job.error ?? "ИИ не смог подготовить подсказку")
        : Boolean(appliedJobId) && appliedJobId === job?.id
          ? "Подсказка добавлена в форму"
          : undefined;
  const isGenerating =
    submitMutation.isPending ||
    (Boolean(jobId) && !job && !jobQuery.isError) ||
    job?.status === "pending" ||
    job?.status === "processing";
  const trimmedLength = input.trim().length;

  return (
    <section className={styles.assistant} aria-labelledby="item-ai-title">
      <div className={styles.assistant__heading}>
        <span className={styles.assistant__icon} aria-hidden="true">
          <Sparkles size={18} />
        </span>
        <div>
          <h3 id="item-ai-title" className={styles.assistant__title}>
            Заполнить с ИИ
          </h3>
          <p className={styles.assistant__hint}>
            Расскажите о вещи своими словами — помощник предложит название,
            описание и категорию.
          </p>
        </div>
      </div>

      <textarea
        className={styles.assistant__input}
        rows={4}
        maxLength={1200}
        value={input}
        disabled={disabled || isGenerating}
        placeholder="Например: красный горный велосипед, катался два сезона, всё работает, на раме есть небольшая царапина"
        onChange={(event) => setInput(event.target.value)}
      />

      <div className={styles.assistant__controls}>
        <span className={styles.assistant__counter}>{input.length}/1200</span>
        <Button
          color="green"
          type="button"
          disabled={disabled || isGenerating || trimmedLength < 10}
          onClick={() => submitMutation.mutate(input.trim())}
        >
          {isGenerating ? "Готовим подсказку..." : "Предложить заполнение"}
        </Button>
      </div>

      {message && (
        <p className={styles.assistant__error} role="alert">
          {message}
        </p>
      )}

      {suggestion && (
        <div className={styles.assistant__result}>
          <div className={styles.assistant__resultHeader}>
            <strong>Готовая подсказка</strong>
            <span>{suggestion.category_name}</span>
          </div>
          <p className={styles.assistant__resultTitle}>{suggestion.title}</p>
          <p className={styles.assistant__description}>
            {suggestion.description}
          </p>
          <p className={styles.assistant__notice}>
            Проверьте факты перед применением — ИИ может ошибаться.
          </p>
          <Button
            color="green"
            type="button"
            disabled={disabled}
            onClick={() => {
              onApply(suggestion);
              setAppliedJobId(job?.id);
            }}
          >
            Применить к объявлению
          </Button>
        </div>
      )}
    </section>
  );
};
