import { memo } from "react";
import { useNavigate } from "react-router-dom";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { Popup } from "../../UI/Popup/Popup";

const LoginComponent = () => {
  const navigate = useNavigate();

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    // Здесь позже будет отправка формы
    console.log("Вход");
  };

  return (
    <Popup>
      <form className={styles.login} onSubmit={handleSubmit} noValidate>
        <div className={styles.login__header}>
          <h2 className={styles.login__title}>Вход</h2>

          <p className={styles.login__description}>
            Войдите в аккаунт, чтобы продолжить
          </p>
        </div>

        <div className={styles.login__fields}>
          <label className={styles.login__label}>
            <span className={styles.login__labelText}>Email</span>

            <input
              className={styles.login__input}
              type="email"
              placeholder="example@mail.ru"
              autoComplete="email"
              required
            />
          </label>

          <label className={styles.login__label}>
            <span className={styles.login__labelText}>Пароль</span>

            <input
              className={styles.login__input}
              type="password"
              placeholder="Введите пароль"
              autoComplete="current-password"
              required
            />
          </label>
        </div>

        <div className={styles.login__buttons}>
          <Button
            color="transparent"
            type="button"
            onClick={() => navigate(-1)}
          >
            Отменить
          </Button>

          <Button color="light" type="submit" centered>
            Войти
          </Button>
        </div>
        <div className={styles.login__register}>
          <span className={styles.login__registerText}>Нет аккаунта?</span>

          <Button
            color="invisible"
            type="button"
            onClick={() => navigate("/register")}
          >
            Зарегистрироваться
          </Button>
        </div>
      </form>
    </Popup>
  );
};

export const Login = memo(LoginComponent);
