import {
  forwardRef,
  memo,
  type ChangeEventHandler,
  type FocusEventHandler,
} from "react";

import styles from "./Styles.module.scss";

type TInput = {
  label?: string;
  className?: string;
  placeholder?: string;
  error?: string;
  type?: "password" | "email" | "text";
  required?: boolean;
  disabled?: boolean;
  textarea?: boolean;
  rows?: number;
  maxLength?: number;
  counter?: string;
  defaultValue?: string;
  value?: string;
  autoComplete?: string;
  name?: string;
  onChange?: ChangeEventHandler<HTMLInputElement | HTMLTextAreaElement>;
  onBlur?: FocusEventHandler<HTMLInputElement | HTMLTextAreaElement>;
};

const InputComponent = forwardRef<
  HTMLInputElement | HTMLTextAreaElement,
  TInput
>(
  (
    {
      label,
      error,
      required = false,
      disabled = false,
      className = "",
      type = "text",
      placeholder,
      textarea = false,
      rows = 4,
      maxLength,
      counter,
      defaultValue,
      value,
      autoComplete,
      name,
      onChange,
      onBlur,
    },
    ref,
  ) => {
    const fieldClassName = [
      styles.input__field,
      textarea ? styles.input__field_textarea : "",
      error ? styles.input__field_error : "",
      className,
    ]
      .filter(Boolean)
      .join(" ");

    const commonProps = {
      className: fieldClassName,
      placeholder,
      maxLength,
      defaultValue,
      value,
      autoComplete,
      name,
      required,
      disabled,
      onChange,
      onBlur,
    };

    return (
      <label className={styles.input}>
        {label && (
          <span
            className={`${styles.input__labelText} ${
              required ? styles.input__labelText_required : ""
            }`}
          >
            {label}
          </span>
        )}

        {textarea ? (
          <textarea
            {...commonProps}
            ref={ref as React.Ref<HTMLTextAreaElement>}
            rows={rows}
          />
        ) : (
          <input
            {...commonProps}
            ref={ref as React.Ref<HTMLInputElement>}
            type={type}
          />
        )}

        {(error || counter) && (
          <span className={styles.input__footer}>
            {error && <span className={styles.input__error}>{error}</span>}
            {counter && (
              <span className={styles.input__counter}>{counter}</span>
            )}
          </span>
        )}
      </label>
    );
  },
);

InputComponent.displayName = "Input";

export const Input = memo(InputComponent);
