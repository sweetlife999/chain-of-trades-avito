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
  autoComplete?: string;
  name?: string;

  onChange?: ChangeEventHandler<
    HTMLInputElement | HTMLTextAreaElement
  >;

  onBlur?: FocusEventHandler<
    HTMLInputElement | HTMLTextAreaElement
  >;
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
      autoComplete,
      name,

      onChange,
      onBlur,

      ...rest
    },
    ref,
  ) => {
    const Tag = textarea ? "textarea" : "input";

    const inputClassName = [
      styles.input__field,
      textarea ? styles.input__field_textarea : "",
      error ? styles.input__field_error : "",
      className,
    ]
      .filter(Boolean)
      .join(" ");

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

        <Tag
          ref={ref as any}
          className={inputClassName}
          type={textarea ? undefined : type}
          placeholder={placeholder}
          rows={textarea ? rows : undefined}
          maxLength={maxLength}
          defaultValue={defaultValue}
          autoComplete={autoComplete}
          name={name}
          required={required}
          disabled={disabled}
          onChange={onChange}
          onBlur={onBlur}
          {...rest}
        />

        {(error || counter) && (
          <span className={styles.input__footer}>
            {error && (
              <span className={styles.input__error}>
                {error}
              </span>
            )}

            {counter && (
              <span className={styles.input__counter}>
                {counter}
              </span>
            )}
          </span>
        )}
      </label>
    );
  },
);

InputComponent.displayName = "Input";

export const Input = memo(InputComponent);