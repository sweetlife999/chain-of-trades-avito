import { memo } from "react";
import clsx from "clsx";

import styles from "./Styles.module.scss";

export type LoaderSize = "small" | "medium" | "large" | "extraLarge";

interface LoaderProps {
  size?: LoaderSize;
  text?: string;
  fullScreen?: boolean;
  className?: string;
}

const LoaderComponent = ({
  size = "medium",
  text,
  fullScreen = false,
  className,
}: LoaderProps) => {
  return (
    <div
      className={clsx(
        styles.loaderWrapper,
        fullScreen && styles.fullScreen,
        className,
      )}
      role="status"
      aria-live="polite"
      aria-label={text ?? "Загрузка"}
    >
      <div
        className={clsx(styles.loader, styles[size])}
        aria-hidden="true"
      >
        <span className={styles.link} />
        <span className={styles.link} />
        <span className={styles.link} />
      </div>

      {text && <span className={styles.text}>{text}</span>}
    </div>
  );
};

export const Loader = memo(LoaderComponent);