import { memo } from "react";
import { Star } from "lucide-react";

import styles from "./Styles.module.scss";

const scale = [1, 2, 3, 4, 5];

const iconSize = { s: 14, m: 20 } as const;

// Склонение отдаём платформе: правила русского счёта она знает лучше, чем ветка на остатках.
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

const RatingComponent = ({
  value,
  count,
  size = "m",
  withValue = true,
}: TRatingProps) => {
  // 4.7 — это 4.7 звезды, а не пять: округление до целой звезды стирает разницу между
  // «почти безупречно» и «безупречно», а её-то и читают в первую очередь.
  const filledWidth = value === null ? 0 : (value / scale.length) * 100;

  return (
    <span className={`${styles.rating} ${styles[`rating_${size}`]}`}>
      <span
        className={styles.rating__stars}
        role="img"
        aria-label={ratingLabel(value, count)}
      >
        <span className={styles.rating__row}>
          {scale.map((star) => (
            <Star key={star} size={iconSize[size]} strokeWidth={1.5} />
          ))}
        </span>

        <span
          className={styles.rating__row_filled}
          style={{ width: `${filledWidth}%` }}
        >
          {scale.map((star) => (
            <Star
              key={star}
              size={iconSize[size]}
              strokeWidth={1.5}
              fill="currentColor"
            />
          ))}
        </span>
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
};

export const Rating = memo(RatingComponent);
