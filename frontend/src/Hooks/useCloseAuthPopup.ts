import { useCallback } from "react";
import { useLocation, useNavigate } from "react-router-dom";

type AuthLocationState = {
  returnTo?: string;
};

export const useCloseAuthPopup = () => {
  const navigate = useNavigate();
  const location = useLocation();

  const closeAuthPopup = useCallback(() => {
    const state = location.state as AuthLocationState | null;

    const returnTo = state?.returnTo?.startsWith("/") ? state.returnTo : "/";

    navigate(returnTo, {
      replace: true,
    });
  }, [location.state, navigate]);

  return closeAuthPopup;
};
