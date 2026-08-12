import { memo, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { z } from "zod";

import styles from "./Styles.module.scss";
import { getExchanges } from "../../../Api/exchanges/exchanges";
import type { TExchangeRatingSlot } from "../../../Api/exchanges/exchanges.types";
import {
  getRatingError,
  getUserRatings,
  rateExchangePartner,
} from "../../../Api/ratings/ratings";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { Button } from "../../UI/Button/Button";
import { Input } from "../../UI/Input/Input";
import { Rating } from "../../UI/Rating/Rating";
import { RatingInput } from "../../UI/Rating/RatingInput";

const pageSize = 20;

const editRatingSchema = z.object({
  score: z.enum(["1", "2", "3", "4", "5"], "Поставьте оценку"),
  comment: z.string().trim(),
});

type TEditRatingForm = z.infer<typeof editRatingSchema>;

type TProfileRatingsProps = {
  userId: string;
  ratingsCount: number;
};

type TEditRatingFormProps = {
  exchangeId: string;
  rating: TExchangeRatingSlot;
  userId: string;
  onCancel: () => void;
  onSaved: () => void;
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(new Date(value));

const formatDeadline = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));

const EditRatingForm = ({
  exchangeId,
  rating,
  userId,
  onCancel,
  onSaved,
}: TEditRatingFormProps) => {
  const queryClient = useQueryClient();
  const {
    formState: { errors },
    handleSubmit,
    register,
    setError,
  } = useForm<TEditRatingForm>({
    resolver: zodResolver(editRatingSchema),
    defaultValues: {
      score: String(rating.score) as TEditRatingForm["score"],
      comment: rating.comment,
    },
  });

  const mutation = useMutation({
    mutationFn: (values: TEditRatingForm) =>
      rateExchangePartner(exchangeId, {
        score: Number(values.score),
        ...(values.comment ? { comment: values.comment } : {}),
      }),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
        queryClient.invalidateQueries({ queryKey: ["users", userId] }),
        queryClient.invalidateQueries({ queryKey: ["ratings", userId] }),
      ]);
      onSaved();
    },
    onError: (error) =>
      setError("root", {
        type: "server",
        message: getRatingError(error, "Не удалось изменить отзыв"),
      }),
  });

  return (
    <form
      className={styles.ratings__editForm}
      noValidate
      onSubmit={handleSubmit((values) => mutation.mutate(values))}
    >
      <RatingInput disabled={mutation.isPending} {...register("score")} />
      {errors.score && (
        <p className={styles.ratings__editError}>{errors.score.message}</p>
      )}

      <Input
        className={styles.ratings__editComment}
        error={errors.comment?.message}
        placeholder="Комментарий к обмену — необязательно"
        rows={3}
        textarea
        {...register("comment")}
      />

      {errors.root && (
        <p className={styles.ratings__editError}>{errors.root.message}</p>
      )}

      <div className={styles.ratings__editButtons}>
        <Button disabled={mutation.isPending} size="s" type="submit">
          {mutation.isPending ? "Сохраняем..." : "Сохранить изменения"}
        </Button>
        <Button
          color="transparent"
          disabled={mutation.isPending}
          size="s"
          type="button"
          onClick={onCancel}
        >
          Отмена
        </Button>
      </div>
    </form>
  );
};

const ProfileRatingsComponent = ({
  userId,
  ratingsCount,
}: TProfileRatingsProps) => {
  const { user: currentUser } = useAuthSelector();
  const [page, setPage] = useState(0);
  const [now, setNow] = useState(() => Date.now());
  const [editingExchangeId, setEditingExchangeId] = useState<string | null>(
    null,
  );
  const [savedExchangeId, setSavedExchangeId] = useState<string | null>(null);
  const totalPages = Math.max(1, Math.ceil(ratingsCount / pageSize));
  const offset = page * pageSize;

  useEffect(() => {
    setPage(0);
    setEditingExchangeId(null);
    setSavedExchangeId(null);
  }, [userId]);

  // Если после обновления профиля оценок стало меньше, не оставляем пользователя
  // на странице, которой больше не существует.
  useEffect(() => {
    if (page >= totalPages) {
      setPage(totalPages - 1);
    }
  }, [page, totalPages]);

  const ratingsQuery = useQuery({
    queryKey: ["ratings", userId, page],
    queryFn: () => getUserRatings(userId, pageSize, offset),
    enabled: ratingsCount > 0,
    retry: false,
  });

  // GET /users/:id/ratings намеренно анонимен и не возвращает автора или exchange_id.
  // Поэтому право редактировать именно СВОЙ отзыв определяем по собственным обменам:
  // сервер уже возвращает там rated_user_id, score и rate_until.
  const exchangesQuery = useQuery({
    queryKey: ["exchanges"],
    queryFn: getExchanges,
    enabled: Boolean(currentUser && currentUser.id !== userId),
    retry: false,
  });

  const ratedExchanges = useMemo(
    () =>
      (exchangesQuery.data ?? [])
        .filter(
          (exchange) =>
            exchange.status === "completed" &&
            exchange.rating?.rated_user_id === userId &&
            exchange.rating.score !== null,
        )
        .sort(
          (left, right) =>
            new Date(right.closed_at ?? right.updated_at).getTime() -
            new Date(left.closed_at ?? left.updated_at).getTime(),
        ),
    [exchangesQuery.data, userId],
  );

  const editableExchanges = ratedExchanges.filter((exchange) => {
    const deadline = new Date(exchange.rating!.rate_until).getTime();
    return Number.isFinite(deadline) && deadline > now;
  });

  // Если профиль оставили открытым, кнопка редактирования сама исчезнет в rate_until.
  useEffect(() => {
    const deadlines = ratedExchanges
      .map((exchange) => new Date(exchange.rating!.rate_until).getTime())
      .filter((deadline) => Number.isFinite(deadline) && deadline > now);

    if (deadlines.length === 0) {
      return;
    }

    const nearestDeadline = Math.min(...deadlines);
    const timerId = window.setTimeout(
      () => setNow(Date.now()),
      Math.max(0, nearestDeadline - Date.now() + 50),
    );

    return () => window.clearTimeout(timerId);
  }, [now, ratedExchanges]);

  if (ratingsCount === 0) {
    return null;
  }

  const ratings = ratingsQuery.data?.ratings ?? [];
  const rangeStart = ratings.length === 0 ? 0 : offset + 1;
  const rangeEnd = Math.min(offset + ratings.length, ratingsCount);

  return (
    <section className={styles.ratings}>
      <div className={styles.ratings__titleRow}>
        <h2 className={styles.ratings__title}>Отзывы</h2>
        <span className={styles.ratings__total}>{ratingsCount}</span>
      </div>

      {editableExchanges.length > 0 && (
        <div className={styles.ratings__ownReviews}>
          {editableExchanges.map((exchange, index) => {
            const rating = exchange.rating!;
            const isEditing = editingExchangeId === exchange.id;
            const multiple = editableExchanges.length > 1;

            return (
              <div className={styles.ratings__ownReview} key={exchange.id}>
                <div className={styles.ratings__ownReviewHeader}>
                  <div className={styles.ratings__ownReviewInfo}>
                    <strong className={styles.ratings__ownReviewTitle}>
                      {multiple ? `Ваш отзыв №${index + 1}` : "Ваш отзыв"}
                    </strong>
                    <div className={styles.ratings__ownReviewSummary}>
                      <Rating size="s" value={rating.score} />
                      {rating.comment && (
                        <span className={styles.ratings__ownReviewComment}>
                          {rating.comment}
                        </span>
                      )}
                    </div>
                    <span className={styles.ratings__ownReviewDeadline}>
                      Можно изменить до {formatDeadline(rating.rate_until)}
                    </span>
                  </div>

                  <Button
                    color={isEditing ? "light" : "green"}
                    size="s"
                    type="button"
                    onClick={() => {
                      setSavedExchangeId(null);
                      setEditingExchangeId(isEditing ? null : exchange.id);
                    }}
                  >
                    {isEditing ? "Скрыть форму" : "Изменить отзыв"}
                  </Button>
                </div>

                {savedExchangeId === exchange.id && !isEditing && (
                  <p className={styles.ratings__editSaved}>Отзыв изменён</p>
                )}

                {isEditing && (
                  <EditRatingForm
                    exchangeId={exchange.id}
                    rating={rating}
                    userId={userId}
                    onCancel={() => setEditingExchangeId(null)}
                    onSaved={() => {
                      setSavedExchangeId(exchange.id);
                      setEditingExchangeId(null);
                    }}
                  />
                )}
              </div>
            );
          })}
        </div>
      )}

      {ratingsQuery.isPending && (
        <p className={styles.ratings__state}>Загрузка отзывов...</p>
      )}

      {ratingsQuery.isError && (
        <p className={`${styles.ratings__state} ${styles.ratings__state_error}`}>
          Не удалось загрузить отзывы
        </p>
      )}

      {!ratingsQuery.isPending && !ratingsQuery.isError && (
        <ul className={styles.ratings__list}>
          {ratings.map((rating, index) => (
            <li
              className={styles.ratings__item}
              key={`${rating.created_at}-${rating.score}-${offset + index}`}
            >
              <div className={styles.ratings__head}>
                <Rating size="s" value={rating.score} withValue={false} />
                <span className={styles.ratings__author}>Участник обмена</span>
                <time
                  className={styles.ratings__date}
                  dateTime={rating.created_at}
                >
                  {formatDate(rating.created_at)}
                </time>
              </div>
              {rating.comment && (
                <p className={styles.ratings__comment}>{rating.comment}</p>
              )}
            </li>
          ))}
        </ul>
      )}

      {totalPages > 1 && (
        <nav aria-label="Пагинация отзывов" className={styles.ratings__pagination}>
          <span className={styles.ratings__paginationSummary}>
            {rangeStart}–{rangeEnd} из {ratingsCount}
          </span>

          <div className={styles.ratings__paginationControls}>
            <Button
              color="transparent"
              disabled={page === 0 || ratingsQuery.isFetching}
              size="s"
              type="button"
              onClick={() => setPage((current) => Math.max(0, current - 1))}
            >
              Назад
            </Button>

            <span className={styles.ratings__paginationCurrent}>
              {page + 1} / {totalPages}
            </span>

            <Button
              color="transparent"
              disabled={page >= totalPages - 1 || ratingsQuery.isFetching}
              size="s"
              type="button"
              onClick={() =>
                setPage((current) => Math.min(totalPages - 1, current + 1))
              }
            >
              Вперёд
            </Button>
          </div>
        </nav>
      )}
    </section>
  );
};

export const ProfileRatings = memo(ProfileRatingsComponent);
