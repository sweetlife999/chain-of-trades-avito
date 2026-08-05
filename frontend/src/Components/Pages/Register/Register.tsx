import { memo } from "react";
import { useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";

import styles from "./Styles.module.scss";

import { Button } from "../../UI/Button/Button";
import { Input } from "../../UI/Input/Input";
import { Popup } from "../../UI/Popup/Popup";

import { registerUser } from "../../../Api/auth/auth";
import { registerSchema, type TRegister } from "../../../Api/auth/auth.types";



const RegisterComponent = () => {
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm<TRegister>({
    resolver: zodResolver(registerSchema),

    defaultValues: {
      nickname: "",
      password: "",
      description: "",
      photo_url: "",
    },
  });

  const registerMutation = useMutation({
    mutationFn: (data: TRegister) => registerUser(data),

    onSuccess: () => {
      navigate("/login");
    },

    onError: () => {
      setError("root", {
        type: "server",
        message: "Не удалось зарегистрироваться",
      });
    },
  });

  const onSubmit = (data: TRegister) => {
    registerMutation.mutate(data);
  };

  return (
    <Popup>
      <form
        className={styles.register}
        onSubmit={handleSubmit(onSubmit)}
        noValidate
      >
        <div className={styles.register__header}>
          <h2 className={styles.register__title}>
            Регистрация
          </h2>

          <p className={styles.register__description}>
            Создайте аккаунт, чтобы продолжить
          </p>

          {errors.root && (
            <span className={styles.register__error}>
              {errors.root.message}
            </span>
          )}
        </div>

        <div className={styles.register__fields}>
          <Input
            label="Nickname"
            required
            type="text"
            placeholder="Введите nickname"
            autoComplete="username"
            error={errors.nickname?.message}
            {...register("nickname")}
          />

          <Input
            label="Пароль"
            required
            type="password"
            placeholder="Введите пароль"
            autoComplete="new-password"
            error={errors.password?.message}
            {...register("password")}
          />

          <Input
            label="Ссылка на фотографию"
            type="text"
            placeholder="https://example.com/photo.jpg"
            autoComplete="url"
            error={errors.photo_url?.message}
            {...register("photo_url")}
          />

          <Input
            textarea
            label="Описание"
            placeholder="Расскажите немного о себе"
            rows={5}
            error={errors.description?.message}
            {...register("description")}
          />
        </div>

        <div className={styles.register__buttons}>
          <Button
            color="transparent"
            type="button"
            onClick={() => navigate(-1)}
          >
            Отменить
          </Button>

          <Button
            color="light"
            type="submit"
            centered
            disabled={registerMutation.isPending}
          >
            {registerMutation.isPending
              ? "Регистрация..."
              : "Зарегистрироваться"}
          </Button>
        </div>

        <div className={styles.register__login}>
          <span className={styles.register__loginText}>
            Уже есть аккаунт?
          </span>

          <Button
            color="invisible"
            type="button"
            onClick={() => navigate("/login")}
          >
            Войти
          </Button>
        </div>
      </form>
    </Popup>
  );
};

export const Register = memo(RegisterComponent);