import { memo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import { getUserRatings } from "../../../Api/ratings/ratings";
import { Button } from "../../UI/Button/Button";
import { Rating } from "../../UI/Rating/Rating";

const pageSize = 20;

type TProfileRatingsProps = {
  userId: string;
  ratingsCount: number;
};

const formatDate = (value: string) =>
  new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(new Date(value));

const ProfileRatingsComponent = ({
  userId,
  ratingsCount,
}: TProfileRatingsProps) => {
  // Растим окно вместо склейки страниц: список короткий, а одна useQuery проще
  // бесконечной прокрутки, которой здесь нечего прокручивать.
  const [limit, setLimit] = useState(pageSize);

  const ratingsQuery = useQuery({
    queryKey: ["ratings", userId, limit],
    queryFn: () => getUserRatings(userId, limit, 0),
    enabled: ratingsCount > 0,
    retry: false,
  });

  if (ratingsCount === 0) {
    return null;
  }

  const ratings = ratingsQuery.data?.ratings ?? [];

  return (
    <section className={styles.ratings}>
      <h2 className={styles.ratings__title}>Отзывы</h2>

      {ratingsQuery.isPending && (
        <p className={styles.ratings__state}>Загрузка отзывов...</p>
      )}

      {ratingsQuery.isError && (
        <p className={`${styles.ratings__state} ${styles.ratings__state_error}`}>
          Не удалось загрузить отзывы
        </p>
      )}

      <ul className={styles.ratings__list}>
        {ratings.map((rating) => (
          <li className={styles.ratings__item} key={`${rating.created_at}-${rating.score}`}>
            <div className={styles.ratings__head}>
              <Rating size="s" value={rating.score} withValue={false} />
              {/* Автора не показываем: его нет в ответе, и это не оформление, а условие. */}
              <span className={styles.ratings__author}>Участник обмена</span>
              <time className={styles.ratings__date} dateTime={rating.created_at}>
                {formatDate(rating.created_at)}
              </time>
            </div>
            {rating.comment && (
              <p className={styles.ratings__comment}>{rating.comment}</p>
            )}
          </li>
        ))}
      </ul>

      {ratings.length < ratingsCount && (
        <Button
          color="transparent"
          disabled={ratingsQuery.isFetching}
          size="s"
          type="button"
          onClick={() => setLimit((current) => current + pageSize)}
        >
          {ratingsQuery.isFetching ? "Загружаем..." : "Показать ещё"}
        </Button>
      )}
    </section>
  );
};

export const ProfileRatings = memo(ProfileRatingsComponent);
