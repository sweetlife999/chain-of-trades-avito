import { forwardRef, memo, useId } from "react";
import { Star } from "lucide-react";

import styles from "./Styles.module.scss";

const scale = [1, 2, 3, 4, 5];

type TRatingInputProps = {
  name: string;
  disabled?: boolean;
  onChange?: React.ChangeEventHandler<HTMLInputElement>;
  onBlur?: React.FocusEventHandler<HTMLInputElement>;
};

// Радиокнопки, а не кнопки с обработчиком: стрелки клавиатуры, фокус, участие в форме и
// восстановление уже выбранного значения браузер отдаёт даром. forwardRef — чтобы сюда
// расстилался register() из React Hook Form, как это делает Input.
const RatingInputComponent = forwardRef<HTMLInputElement, TRatingInputProps>(
  ({ name, disabled, onChange, onBlur }, ref) => {
    const groupId = useId();

    return (
      <span className={styles.input} role="radiogroup" aria-label="Оценка от 1 до 5">
        {scale.map((star) => {
          const id = `${groupId}-${star}`;
          return (
            <span className={styles.input__star} key={star}>
              <input
                className={styles.input__control}
                type="radio"
                id={id}
                name={name}
                value={star}
                ref={ref}
                disabled={disabled}
                onChange={onChange}
                onBlur={onBlur}
              />
              <label className={styles.input__label} htmlFor={id}>
                <Star size={28} strokeWidth={1.5} fill="currentColor" />
                <span className={styles.input__text}>{star} из 5</span>
              </label>
            </span>
          );
        })}
      </span>
    );
  },
);

RatingInputComponent.displayName = "RatingInput";

export const RatingInput = memo(RatingInputComponent);
