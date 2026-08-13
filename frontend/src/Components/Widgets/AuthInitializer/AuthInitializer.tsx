import axios from "axios";
import { useEffect, useRef, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import { getMe } from "../../../Api/auth/auth";
import { useAuthDispatch } from "../../../Hooks/useAuthDispatch";
import { logoutState, setUserState } from "../../../Store/authSlice";
import { Button } from "../../UI/Button/Button";
import { Loader } from "../../UI/Loader/Loader";
import { useMascot } from "../../../Hooks/useMascot";

type AuthInitializerProps = {
  children: ReactNode;
};

export const AuthInitializer = ({ children }: AuthInitializerProps) => {
  const dispatch = useAuthDispatch();
  const { reactTo } = useMascot();
  const previousLevelRef = useRef<number | null>(null);

  const {
    data: user,
    error,
    isPending,
    refetch,
  } = useQuery({
    queryKey: ["auth", "me"],
    queryFn: getMe,
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
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
      const previousLevel = previousLevelRef.current;
      previousLevelRef.current = user.experience.level;
      dispatch(setUserState(user));

      if (previousLevel !== null && user.experience.level > previousLevel) {
        reactTo("LEVEL_UP");
      }
      return;
    }

    if (status === 401) {
      previousLevelRef.current = null;
      dispatch(logoutState());
    }
  }, [dispatch, reactTo, status, user]);

  if (isPending) {
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
