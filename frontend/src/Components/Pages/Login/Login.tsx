import { memo } from "react";
import { useNavigate } from "react-router-dom";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { Popup } from "../../UI/Popup/Popup";
import { formSchema, type FormState } from "./shemaLogin";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { login } from "../../../Api/auth/auth";
import { useAuthDispatch } from "../../../Hooks/useAuthDispatch";
import { setUserState } from "../../../Store/authSlice";
import { Input } from "../../UI/Input/Input";
import type { TAuthenticatedUser } from "../../../Api/auth/auth.types";

const LoginComponent = () => {
  const navigate = useNavigate();
  const dispatch = useAuthDispatch();

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm<FormState>({
    resolver: zodResolver(formSchema),
    
  });

const loginMutation = useMutation({
  mutationFn: (data: FormState) => login(data),

  onSuccess: (data: TAuthenticatedUser) => {
    dispatch(setUserState(data));
    navigate("/profile");
  },

  onError: () => {
    setError("root", {
      type: "server",
      message: "Неверный nickname или пароль",
    });
  },
});

  const onSubmit = (data: FormState) => {
    loginMutation.mutate(data);
  };

  return (
    <Popup>
      <form
        className={styles.login}
        onSubmit={handleSubmit(onSubmit)}
        noValidate
      >
        <div className={styles.login__header}>
          <h2 className={styles.login__title}>Вход</h2>

          <p className={styles.login__description}>
            Войдите в аккаунт, чтобы продолжить
          </p>
          {errors.root && (
            <span className={styles.login__error}>{errors.root.message}</span>
          )}
        </div>

        <div className={styles.login__fields}>
          <label className={styles.login__label}>
            <span className={styles.login__labelText}>Nickname</span>

            <input
              className={styles.login__input}
              type="text"
              placeholder="nickname"
              // autoComplete="email"
              {...register("nickname")}
            />
            {errors.nickname && (
              <span className={styles.login__error}>
                {errors.nickname.message}
              </span>
            )}
          </label>

          <Input
            label="Пароль"
            type="password"
            autoComplete="current-password"
            placeholder="Введите пароль"
            error={errors.password?.message}
            {...register("password")}
          />
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
