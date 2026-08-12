import { memo, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { z } from "zod";

import styles from "./Styles.module.scss";
import {
  getRatingError,
  rateExchangePartner,
} from "../../../../Api/ratings/ratings";
import type { TExchangeRatingSlot } from "../../../../Api/exchanges/exchanges.types";
import { Button } from "../../../UI/Button/Button";
import { Input } from "../../../UI/Input/Input";
import { RatingInput } from "../../../UI/Rating/RatingInput";

const ratingFormSchema = z.object({
  score: z.enum(["1", "2", "3", "4", "5"], "Поставьте оценку"),
  // В Swagger комментарий необязательный и отдельный лимит длины не задан.
  comment: z.string().trim(),
});

type TRatingForm = z.infer<typeof ratingFormSchema>;

type TExchangeRatingProps = {
  exchangeId: string;
  partnerNickname: string;
  rating: TExchangeRatingSlot;
};

const formatDeadline = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));

const ExchangeRatingComponent = ({
  exchangeId,
  partnerNickname,
  rating,
}: TExchangeRatingProps) => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const alreadyRated = rating.score !== null;
  const deadlineTimestamp = useMemo(
    () => new Date(rating.rate_until).getTime(),
    [rating.rate_until],
  );
  const [windowClosed, setWindowClosed] = useState(
    () => !Number.isFinite(deadlineTimestamp) || deadlineTimestamp <= Date.now(),
  );

  // Форма должна исчезнуть ровно в момент rate_until, который прислал сервер.
  useEffect(() => {
    if (!Number.isFinite(deadlineTimestamp)) {
      setWindowClosed(true);
      return;
    }

    const remaining = deadlineTimestamp - Date.now();
    if (remaining <= 0) {
      setWindowClosed(true);
      return;
    }

    setWindowClosed(false);
    const timerId = window.setTimeout(() => setWindowClosed(true), remaining);

    return () => window.clearTimeout(timerId);
  }, [deadlineTimestamp]);

  const {
    formState: { errors },
    handleSubmit,
    register,
    setError,
  } = useForm<TRatingForm>({
    resolver: zodResolver(ratingFormSchema),
    defaultValues: {
      comment: "",
    },
  });

  const rateMutation = useMutation({
    mutationFn: (values: TRatingForm) =>
      rateExchangePartner(exchangeId, {
        score: Number(values.score),
        ...(values.comment ? { comment: values.comment } : {}),
      }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
        queryClient.invalidateQueries({
          queryKey: ["users", rating.rated_user_id],
        }),
        queryClient.invalidateQueries({
          queryKey: ["ratings", rating.rated_user_id],
        }),
      ]);
    },
    onError: (error) =>
      setError("root", {
        type: "server",
        message: getRatingError(error, "Не удалось сохранить оценку"),
      }),
  });

  const hasSubmittedRating = alreadyRated || rateMutation.isSuccess;

  if (hasSubmittedRating) {
    return (
      <div className={styles.rating}>
        <h3 className={styles.rating__title}>Отзыв оставлен</h3>
        <p className={styles.rating__hint}>
          {windowClosed
            ? `Вы уже оставили отзыв о ${partnerNickname}. Срок его изменения истёк.`
            : `Вы уже оставили отзыв о ${partnerNickname}. Если хотите изменить его, перейдите в профиль этого пользователя.`}
        </p>

        {!windowClosed && (
          <Button
            size="s"
            type="button"
            onClick={() => navigate(`/profile/${rating.rated_user_id}`)}
          >
            Перейти в профиль {partnerNickname}
          </Button>
        )}
      </div>
    );
  }

  if (windowClosed) {
    return (
      <p className={styles.rating__closed}>
        Срок оценки истёк: отзыв можно оставить только в течение 14 дней после
        завершения обмена.
      </p>
    );
  }

  return (
    <form
      className={styles.rating}
      noValidate
      onSubmit={handleSubmit((values) => rateMutation.mutate(values))}
    >
      <h3 className={styles.rating__title}>{`Оцените ${partnerNickname}`}</h3>
      <p className={styles.rating__hint}>
        {partnerNickname} передал вам свою вещь. Оставить отзыв можно до{" "}
        {formatDeadline(rating.rate_until)}. После первой отправки изменить его
        можно будет только в профиле этого пользователя и только до этой даты.
      </p>

      <RatingInput disabled={rateMutation.isPending} {...register("score")} />
      {errors.score && (
        <p className={styles.rating__error}>{errors.score.message}</p>
      )}

      <Input
        className={styles.rating__comment}
        error={errors.comment?.message}
        placeholder="Комментарий к обмену — необязательно"
        rows={3}
        textarea
        {...register("comment")}
      />

      {errors.root && <p className={styles.rating__error}>{errors.root.message}</p>}

      <Button disabled={rateMutation.isPending} size="s" type="submit">
        {rateMutation.isPending ? "Сохраняем..." : "Оставить отзыв"}
      </Button>
    </form>
  );
};

export const ExchangeRating = memo(ExchangeRatingComponent);
