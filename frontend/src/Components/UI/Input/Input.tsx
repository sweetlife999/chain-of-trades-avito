import  { forwardRef, memo } from "react";
import styles from "./Styles.module.scss";

type TInput = {
  label?: string;
  className?: string;
  placeholder?: string;
  error?: string;
  type?: "password" | "email" | "text";
  required?: boolean;
  textarea?: boolean;
  rows?: number;
  maxLength?: number;
  counter?: string;
  defaultValue?: string;
  autoComplete?: string;
  name?: string;
};

const InputComponent = forwardRef<HTMLInputElement | HTMLTextAreaElement, TInput>(
  (
    {
      label,
      error,
      required = false,
      className = "",
      type = "text",
      placeholder,
      textarea = false,
      rows = 4,
      maxLength,
      counter,
      defaultValue,
      autoComplete,
      // onChange,
      // onBlur,
      name,
      ...rest // register может передать ещё что-то
    },
    ref,
  ) => {
    const Tag = textarea ? "textarea" : "input";
    
    return (
      <label className={styles.form__label}>
        {label && (
          <span className={`${styles.form__title} ${required && styles.form__title_required}`}>
            {label}
          </span>
        )}
        
        <Tag
          ref={ref as any}
          className={`${styles.form__input} ${textarea ? styles.form__input_comm : ""} ${error ? styles.form__input_error : ""} ${className}`}
          type={!textarea ? type : undefined}
          placeholder={placeholder}
          maxLength={maxLength}
          rows={textarea ? rows : undefined}
          defaultValue={defaultValue}
          autoComplete={autoComplete}
          // onChange={onChange}
          // onBlur={onBlur}
          name={name}
          {...rest}
        />
        
        <div className={styles.form__label_cover}>
          {error && <span className={styles.form__error}>{error}</span>}
          {counter && <span className={styles.form__counter}>{counter}</span>}
        </div>
      </label>
    );
  },
);

InputComponent.displayName = "Input";

export const Input = memo(InputComponent);