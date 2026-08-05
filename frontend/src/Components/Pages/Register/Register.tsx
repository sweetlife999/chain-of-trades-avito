import { memo } from "react";
import { useNavigate } from "react-router-dom";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { Popup } from "../../UI/Popup/Popup";

const RegisterComponent = () => {
  const navigate = useNavigate();

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    // Здесь позже будет отправка формы
    console.log("Вход");
  };

  return (
    <Popup>
      <form className={styles.register} onSubmit={handleSubmit} noValidate>
        <div className={styles.register__header}>
          <h2 className={styles.register__title}>Регистрация</h2>

          <p className={styles.register__description}>
            Зарегистрируйтесь, чтобы продолжить
          </p>
        </div>

        <div className={styles.register__fields}>
          <label className={styles.register__label}>
            <span className={styles.register__labelText}>Email</span>

            <input
              className={styles.register__input}
              type="email"
              placeholder="example@mail.ru"
              autoComplete="email"
              required
            />
          </label>

          <label className={styles.register__label}>
            <span className={styles.register__labelText}>Пароль</span>

            <input
              className={styles.register__input}
              type="password"
              placeholder="Введите пароль"
              autoComplete="new-password"
              required
            />
          </label>

          <label className={styles.register__label}>
            <span className={styles.register__labelText}>
              Подтвердите пароль
            </span>

            <input
              className={styles.register__input}
              type="password"
              placeholder="Подтвердите пароль"
              autoComplete="new-password"
              required
            />
          </label>
        </div>

        <div className={styles.register__buttons}>
          <Button
            color="transparent"
            type="button"
            onClick={() => navigate(-2)}
          >
            Отменить
          </Button>

          <Button color="light" type="submit" centered>
            Зарегистрироваться
          </Button>
        </div>
      </form>
    </Popup>
  );
};

export const Register = memo(RegisterComponent);
