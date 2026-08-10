import { LogIn, UserPlus } from "lucide-react";
import { Link } from "react-router-dom";

import styles from "./Styles.module.scss";

type TAuthRequiredStateProps = {
  description: string;
  returnTo: string;
  title: string;
};

export const AuthRequiredState = ({
  description,
  returnTo,
  title,
}: TAuthRequiredStateProps) => (
  <div className={styles.authState}>
    <span className={styles.authState__icon} aria-hidden="true">
      <LogIn className={styles.authState__iconGraphic} />
    </span>
    <h2 className={styles.authState__title}>{title}</h2>
    <p className={styles.authState__description}>{description}</p>
    <div className={styles.authState__actions}>
      <Link
        className={styles.authState__primary}
        state={{ closeTo: returnTo, from: returnTo }}
        to="/login"
      >
        <LogIn aria-hidden="true" size={18} />
        Войти в аккаунт
      </Link>
      <Link
        className={styles.authState__secondary}
        state={{ closeTo: returnTo, from: returnTo }}
        to="/register"
      >
        <UserPlus aria-hidden="true" size={18} />
        Зарегистрироваться
      </Link>
    </div>
  </div>
);
