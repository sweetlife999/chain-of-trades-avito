import { memo, useEffect, type ReactNode } from "react";
import styles from "./Styles.module.scss";
import CloseIcon from "/src/Assets/close.svg";
import { useLocation, useNavigate } from "react-router-dom";

type PopupProps = {
  children: ReactNode;
};

const PopupComponent = ({ children }: PopupProps) => {
  const navigate = useNavigate();
  const location = useLocation()

  const closePopup = () => {
    const link = location.pathname.includes('/login') ? -1 : -2
    navigate(link);
  };

  const handleClickAway = (evt: KeyboardEvent) => {
    if (evt.key === "Escape") {
      closePopup();
    }
  };

  useEffect(() => {
    window.addEventListener("keydown", handleClickAway);

    return () => {
      window.removeEventListener("keydown", handleClickAway);
    };
  }, []);

  const handleOverlay = (evt: React.MouseEvent<HTMLDivElement>) => {
    if (evt.target === evt.currentTarget) {
      closePopup();
    }
  };

  return (
    <div
      className={styles.popup__overlay}
      onClick={handleOverlay}
    >
      <div className={styles.popup}>
        <img
          className={styles.popup__close}
          onClick={closePopup}
          src={CloseIcon}
        />

        {children}
      </div>
    </div>
  );
};

export const Popup = memo(PopupComponent);