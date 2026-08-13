import { useCallback } from "react";
import { useNavigate } from "react-router-dom";

import { logout } from "../Api/auth/auth";
import { logoutState } from "../Store/authSlice";
import { useAuthDispatch } from "./useAuthDispatch";
import { tutorialStopped } from "../Features/Tutorial/tutorialSlice";

export const useLogout = () => {
  const navigate = useNavigate();
  const dispatch = useAuthDispatch();

  return useCallback(async () => {
    try {
      await logout();

      dispatch(logoutState());
      dispatch(tutorialStopped());
      navigate("/");
    } catch (error) {
      console.error("Не удалось выйти", error);
    }
  }, [dispatch, navigate]);
};
