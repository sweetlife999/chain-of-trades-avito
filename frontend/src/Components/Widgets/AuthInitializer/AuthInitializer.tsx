import axios from "axios";
import {
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { useQuery } from "@tanstack/react-query";

import { getMe } from "../../../Api/auth/auth";
import {
  logoutState,
  setUserState,
} from "../../../Store/authSlice";
import { useAuthDispatch } from "../../../Hooks/useAuthDispatch";

import { Loader } from "../../UI/Loader/Loader";
import { Button } from "../../UI/Button/Button";

type AuthInitializerProps = {
  children: ReactNode;
};

export const AuthInitializer = ({
  children,
}: AuthInitializerProps) => {
  const dispatch = useAuthDispatch();

  /*
   * Показывает, что результат GET /auth/me
   * уже применён к Redux.
   */
  const [isInitialized, setIsInitialized] = useState(false);

  const {
    data: user,
    error,
    isFetching,
    refetch,
  } = useQuery({
    queryKey: ["auth", "me"],
    queryFn: getMe,

    /*
     * 401 означает, что cookie отсутствует или истекла.
     * Повторять такой запрос бессмысленно.
     *
     * Временные сетевые ошибки пробуем повторить два раза.
     */
    retry: (failureCount, requestError) => {
      if (
        axios.isAxiosError(requestError) &&
        requestError.response?.status === 401
      ) {
        return false;
      }

      return failureCount < 2;
    },

    /*
     * Проверка нужна при запуске приложения,
     * а не при каждом возвращении во вкладку.
     */
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (user) {
      dispatch(setUserState(user));
      setIsInitialized(true);

      return;
    }

    if (!error) {
      return;
    }

    const status = axios.isAxiosError(error)
      ? error.response?.status
      : undefined;

    /*
     * Только 401 означает, что пользователь
     * действительно не авторизован.
     */
    if (status === 401) {
      dispatch(logoutState());
      setIsInitialized(true);
    }
  }, [user, error, dispatch]);

  /*
   * Пока сессия проверяется, Header не рендерится.
   * Поэтому кнопка "Вход и регистрация" не успеет мигнуть.
   */
  if (!isInitialized) {
    const status = axios.isAxiosError(error)
      ? error.response?.status
      : undefined;

    /*
     * Не считаем пользователя разлогиненным,
     * если backend временно недоступен или вернул 500.
     */
    if (error && status !== 401 && !isFetching) {
      return (
        <div role="alert">
          <p>Не удалось проверить авторизацию.</p>

          <Button
            type="button"
            color="light"
            onClick={() => {
              void refetch();
            }}
          >
            Повторить
          </Button>
        </div>
      );
    }

    return (
      <Loader
        fullScreen
        size="large"
        text="Проверяем авторизацию..."
      />
    );
  }

  return children;
};