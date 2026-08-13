import React, { memo, type JSX, type MouseEventHandler } from "react";
import styles from "./Styles.module.scss";

type TButton = {
  children?: React.ReactNode;
  color?: "green" | "light" | "dark" | "danger" | "transparent" | "invisible";
  onClick?: MouseEventHandler<HTMLButtonElement>;
  size?: "s" | "m" | 'l';
  centered?: boolean;
  type?: "button" | "submit";
  disabled?: boolean;
  className?: string;
  mascotAnchor?: string;
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
  mascotAnchor,
}: TButton): JSX.Element => {
  const buttonClassName = [
    styles.button,
    styles[`button_${color}`],
    styles[`button_${size}`],
    centered ? styles.button_centered : "",
    className ?? "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button
      className={buttonClassName}
      onClick={onClick}
      type={type}
      disabled={disabled}
      data-mascot-anchor={mascotAnchor}
    >
      {children}
    </button>
  );
};

export const Button = memo(ButtonComponent);
