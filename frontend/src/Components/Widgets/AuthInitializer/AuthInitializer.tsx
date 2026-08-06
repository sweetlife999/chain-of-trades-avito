import axios from "axios";
import { useEffect, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import { getMe } from "../../../Api/auth/auth";
import { useAuthDispatch } from "../../../Hooks/useAuthDispatch";
import { logoutState, setUserState } from "../../../Store/authSlice";
import { Button } from "../../UI/Button/Button";
import { Loader } from "../../UI/Loader/Loader";

type AuthInitializerProps = {
  children: ReactNode;
};

export const AuthInitializer = ({ children }: AuthInitializerProps) => {
  const dispatch = useAuthDispatch();

  const {
    data: user,
    error,
    isPending,
    isFetching,
    refetch,
  } = useQuery({
    queryKey: ["auth", "me"],
    queryFn: getMe,
    retry: (failureCount, requestError) => {
      if (
        axios.isAxiosError(requestError) &&
        requestError.response?.status === 401
      ) {
        return false;
      }

      return failureCount < 2;
    },
    refetchOnWindowFocus: false,
  });

  const status = axios.isAxiosError(error)
    ? error.response?.status
    : undefined;

  useEffect(() => {
    if (user) {
      dispatch(setUserState(user));
      return;
    }

    if (status === 401) {
      dispatch(logoutState());
    }
  }, [dispatch, status, user]);

  if (isPending || isFetching) {
    return (
      <Loader
        fullScreen
        size="large"
        text="Проверяем авторизацию..."
      />
    );
  }

  if (error && status !== 401) {
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

  return children;
};
