import { memo } from "react";
import { Star } from "lucide-react";

import styles from "./Styles.module.scss";

const scale = [1, 2, 3, 4, 5];

const iconSize = { s: 14, m: 20 } as const;

const pluralRules = new Intl.PluralRules("ru-RU");
const ratingForms: Record<Intl.LDMLPluralRule, string> = {
  one: "оценка",
  few: "оценки",
  many: "оценок",
  other: "оценок",
  zero: "оценок",
  two: "оценки",
};

type TRatingProps = {
  /** null — оценок ещё нет. Это не ноль, и выглядеть как ноль не должно. */
  value: number | null;
  /** Сколько оценок собрано. Не передан — подпись не показывается. */
  count?: number;
  size?: "s" | "m";
  /** false — только звёзды: в строке отзыва число рядом с ними ничего не добавляет. */
  withValue?: boolean;
};

const ratingLabel = (value: number | null, count?: number) => {
  if (value === null) {
    return "Оценок пока нет";
  }
  if (count === undefined) {
    return `Рейтинг ${value.toFixed(1)} из 5`;
  }
  return `Рейтинг ${value.toFixed(1)} из 5 по ${count} ${ratingForms[pluralRules.select(count)]}`;
};

const getStarFill = (value: number | null, star: number) => {
  if (value === null) {
    return 0;
  }

  // Каждая звезда заполняется отдельно. Поэтому ровно 4.0 всегда означает
  // четыре целые золотые звезды и одну пустую, а 4.7 — четыре целые и 70% пятой.
  return Math.max(0, Math.min(1, value - (star - 1))) * 100;
};

const RatingComponent = ({
  value,
  count,
  size = "m",
  withValue = true,
}: TRatingProps) => (
  <span className={`${styles.rating} ${styles[`rating_${size}`]}`}>
    <span
      className={styles.rating__stars}
      role="img"
      aria-label={ratingLabel(value, count)}
    >
      {scale.map((star) => {
        const fill = getStarFill(value, star);

        return (
          <span className={styles.rating__star} key={star}>
            <Star
              className={styles.rating__starEmpty}
              size={iconSize[size]}
              strokeWidth={1.5}
            />
            {fill > 0 && (
              <span
                aria-hidden="true"
                className={styles.rating__starFill}
                style={{ width: `${fill}%` }}
              >
                <Star
                  size={iconSize[size]}
                  strokeWidth={1.5}
                  fill="currentColor"
                />
              </span>
            )}
          </span>
        );
      })}
    </span>

    {withValue && (
      <span className={styles.rating__value}>
        {value === null ? "—" : value.toFixed(1)}
      </span>
    )}

    {count !== undefined && (
      <span className={styles.rating__count}>
        {value === null
          ? "оценок пока нет"
          : `${count} ${ratingForms[pluralRules.select(count)]}`}
      </span>
    )}
  </span>
);

export const Rating = memo(RatingComponent);
