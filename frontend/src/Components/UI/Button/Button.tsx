import React, { memo, type JSX, type MouseEventHandler } from "react";
import styles from "./Styles.module.scss";

type TButton = {
  children?: React.ReactNode;
  color?: "green" | "light" | "dark" | "transparent" | "invisible";
  onClick?: MouseEventHandler<HTMLButtonElement>;
  size?: "s" | "m" | 'l';
  centered?: boolean;
  type?: "button" | "submit";
  disabled?: boolean;
  className?: string;
};

const ButtonComponent = ({
  children,
  color = "green",
  onClick,
  size = "m",
  centered,
  type = "button",
  disabled,
  className,
}: TButton): JSX.Element => {
  return (
    <button
      className={`${styles.button} ${styles[`button_${color}`]} ${styles[`button_${size}`]} ${centered && styles.button_centered} ${className}`}
      onClick={onClick}
      type={type}
      disabled={disabled}
    >
      {children}
    </button>
  );
};

export const Button = memo(ButtonComponent);
