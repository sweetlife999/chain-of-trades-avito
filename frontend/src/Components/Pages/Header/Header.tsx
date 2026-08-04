import { memo } from "react";
import AvitoLogo from "/avito-logo.svg";
import styles from "./Styles.module.scss";

const HeaderComponent = () => {
  return (
    <header className={styles.header}>
      <div className={styles.header__wrapp}>
        <img className={styles.header__logo} src={AvitoLogo} />
        <h1>This Header</h1>
      </div>
    </header>
  );
};

export const Header = memo(HeaderComponent);
