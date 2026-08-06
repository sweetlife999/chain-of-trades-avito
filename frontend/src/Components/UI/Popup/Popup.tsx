import { memo, useEffect, type MouseEvent, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import CloseIcon from "/src/Assets/close.svg?react";
import styles from "./Styles.module.scss";

type PopupProps = {
  children: ReactNode;
};

const PopupComponent = ({ children }: PopupProps) => {
  const navigate = useNavigate();
  const location = useLocation();

  const closePopup = () => {
    navigate(location.pathname.includes("/login") ? -1 : -2);
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        navigate(location.pathname.includes("/login") ? -1 : -2);
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [location.pathname, navigate]);

  const handleOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      closePopup();
    }
  };

  return (
    <div className={styles.popup__overlay} onClick={handleOverlayClick}>
      <div className={styles.popup}>
        <CloseIcon className={styles.popup__close} onClick={closePopup} />
        {children}
      </div>
    </div>
  );
};

export const Popup = memo(PopupComponent);
