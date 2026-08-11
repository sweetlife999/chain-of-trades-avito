import { memo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
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

const maxCommentLength = 2000;

// Балл остаётся строкой, какой его и отдаёт радиокнопка: одна конвертация на отправке
// дешевле, чем coerce, из-за которого у схемы расходятся вход и выход.
const ratingFormSchema = z.object({
  score: z.enum(["1", "2", "3", "4", "5"], "Поставьте оценку"),
  comment: z.string().trim().max(maxCommentLength),
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
  }).format(new Date(value));

const ExchangeRatingComponent = ({
  exchangeId,
  partnerNickname,
  rating,
}: TExchangeRatingProps) => {
  const queryClient = useQueryClient();
  const alreadyRated = rating.score !== null;
  // Срок считает сервер и присылает готовым моментом: свои часы у клиента разошлись бы
  // с теми, по которым откажет API. Читаем его один раз при открытии экрана, а не в
  // каждом рендере: иначе форма могла бы исчезнуть прямо под руками пишущего.
  const [windowClosed] = useState(
    () => new Date(rating.rate_until).getTime() < Date.now(),
  );

  const {
    formState: { errors },
    handleSubmit,
    register,
    setError,
  } = useForm<TRatingForm>({
    resolver: zodResolver(ratingFormSchema),
    defaultValues: {
      score: rating.score === null ? undefined : (String(rating.score) as TRatingForm["score"]),
      comment: rating.comment,
    },
  });

  const rateMutation = useMutation({
    mutationFn: (values: TRatingForm) =>
      rateExchangePartner(exchangeId, Number(values.score), values.comment),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["ratings"] }),
      ]);
    },
    onError: (error) =>
      setError("root", {
        type: "server",
        message: getRatingError(error, "Не удалось сохранить оценку"),
      }),
  });

  if (windowClosed) {
    return (
      <p className={styles.rating__closed}>
        {alreadyRated
          ? `Вы оценили ${partnerNickname}. Срок изменения оценки истёк.`
          : `Срок оценки истёк — обмен завершился больше двух недель назад.`}
      </p>
    );
  }

  return (
    <form
      className={styles.rating}
      noValidate
      onSubmit={handleSubmit((values) => rateMutation.mutate(values))}
    >
      <h3 className={styles.rating__title}>
        {alreadyRated ? "Ваша оценка" : `Оцените ${partnerNickname}`}
      </h3>
      <p className={styles.rating__hint}>
        {partnerNickname} передал вам свою вещь. Оценку можно изменить до{" "}
        {formatDeadline(rating.rate_until)}.
      </p>

      <RatingInput disabled={rateMutation.isPending} {...register("score")} />
      {errors.score && (
        <p className={styles.rating__error}>{errors.score.message}</p>
      )}

      <Input
        className={styles.rating__comment}
        counter={`до ${maxCommentLength} символов`}
        error={errors.comment?.message}
        maxLength={maxCommentLength}
        placeholder="Что стоит знать другим участникам? Необязательно"
        rows={3}
        textarea
        {...register("comment")}
      />

      {errors.root && <p className={styles.rating__error}>{errors.root.message}</p>}

      <Button disabled={rateMutation.isPending} size="s" type="submit">
        {rateMutation.isPending
          ? "Сохраняем..."
          : alreadyRated
            ? "Изменить оценку"
            : "Оценить"}
      </Button>

      {rateMutation.isSuccess && (
        <p className={styles.rating__saved}>Оценка сохранена</p>
      )}
    </form>
  );
};

export const ExchangeRating = memo(ExchangeRatingComponent);
