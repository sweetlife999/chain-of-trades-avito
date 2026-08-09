import {
  memo,
  useCallback,
  useEffect,
  type MouseEvent,
  type ReactNode,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";

import CloseIcon from "/src/Assets/close.svg";
import styles from "./Styles.module.scss";

type PopupProps = {
  children: ReactNode;
};

const PopupComponent = ({ children }: PopupProps) => {
  const navigate = useNavigate();
  const location = useLocation();
  const locationState = location.state as { closeTo?: string } | null;
  const closeTo = locationState?.closeTo;

  const closePopup = useCallback(() => {
    if (closeTo) {
      navigate(closeTo, { replace: true });
      return;
    }

    navigate(location.pathname.includes("/login") ? -1 : -2);
  }, [closeTo, location.pathname, navigate]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        closePopup();
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [closePopup]);

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      closePopup();
    }
  };

  return (
    <div className={styles.popup__overlay} onClick={handleOverlayClick}>
      <div className={styles.popup}>
        <img className={styles.popup__close} onClick={closePopup} src={CloseIcon} />
        {children}
      </div>
    </div>
  );
};

export const Popup = memo(PopupComponent);
