import { memo, useEffect, useRef } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useForm } from "react-hook-form";
import { useMutation } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";

import styles from "./Styles.module.scss";

import { Button } from "../../UI/Button/Button";
import { Input } from "../../UI/Input/Input";
import { Popup } from "../../UI/Popup/Popup";

import { registerAndLogin } from "../../../Api/auth/auth";
import {
  registerSchema,
  type TAuthenticatedUser,
  type TRegister,
} from "../../../Api/auth/auth.types";
import { useAuthDispatch } from "../../../Hooks/useAuthDispatch";
import { setUserState } from "../../../Store/authSlice";
import { useMascot } from "../../../Hooks/useMascot";

const RegisterComponent = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const dispatch = useAuthDispatch();
  const { reactTo, reset } = useMascot();
  const authenticationSucceededRef = useRef(false);
  const locationState = location.state as
    | { closeTo?: string; from?: string }
    | null;
  const destination = locationState?.from ?? "/profile";

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
    mutationFn: (data: TRegister) => registerAndLogin(data),

    onSuccess: (data: TAuthenticatedUser) => {
      authenticationSucceededRef.current = true;
      reactTo("FORM_SUCCESS");
      dispatch(setUserState(data));
      navigate(destination, { replace: true });
    },

    onError: () => {
      reactTo("FORM_ERROR");
      setError("root", {
        type: "server",
        message: "Не удалось зарегистрироваться",
      });
    },
  });

  const onSubmit = (data: TRegister) => {
    reactTo("FORM_SUBMIT");
    registerMutation.mutate(data);
  };

  useEffect(() => {
    reactTo("AUTH_FORM_READY");

    return () => {
      if (!authenticationSucceededRef.current) {
        reset();
      }
    };
  }, [reactTo, reset]);

  const closeForm = () => {
    if (locationState?.closeTo) {
      navigate(locationState.closeTo, { replace: true });
      return;
    }

    navigate(-1);
  };

  return (
    <Popup>
      <form
        className={styles.register}
        onSubmit={handleSubmit(onSubmit, () => reactTo("FORM_ERROR"))}
        noValidate
      >
        <div className={styles.register__header}>
          <h2 className={styles.register__title}>Регистрация</h2>

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
            label="Никнейм"
            required
            type="text"
            placeholder="Ваш никнейм"
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
            className={styles.register__button}
            color="transparent"
            type="button"
            onClick={closeForm}
          >
            Отменить
          </Button>

          <Button
            className={styles.register__button}
            color="light"
            type="submit"
            centered
            disabled={registerMutation.isPending}
            mascotAnchor="form-submit"
          >
            {registerMutation.isPending
              ? "Регистрация..."
              : "Зарегистрироваться"}
          </Button>
        </div>

        <div className={styles.register__login}>
          <span className={styles.register__loginText}>Уже есть аккаунт?</span>

          <Button
            color="invisible"
            type="button"
            onClick={() =>
              navigate("/login", {
                state: {
                  closeTo: locationState?.closeTo,
                  from: destination,
                },
              })
            }
          >
            Войти
          </Button>
        </div>
      </form>
    </Popup>
  );
};

export const Register = memo(RegisterComponent);
