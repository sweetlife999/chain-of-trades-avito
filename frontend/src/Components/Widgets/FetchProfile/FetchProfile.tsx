import { memo } from "react";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { Link } from "react-router-dom";
import { Button } from "../../UI/Button/Button";
import { ProfilePreview } from "../ProfilePreview/ProfilePreview";
import styles from "./Styles.module.scss";

const FetchProfileComponent = () => {
  const { isAuth, user } = useAuthSelector();

  if (isAuth && user) {
    return <ProfilePreview user={user} />;
  }

  return (
    <Link className={styles.authLink} to="/login">
      <Button className={styles.authLink__button} size="m" color="transparent">
        <span className={styles.authLink__label}>Вход и регистрация</span>
        <span className={styles.authLink__labelCompact}>Войти</span>
      </Button>
    </Link>
  );
};

export const FetchProfile = memo(FetchProfileComponent);
