import { memo } from "react";
import AvitoLogo from "/src/Assets/avito-logo.svg?react";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { Link, NavLink } from "react-router-dom";
import {
  ArrowLeftRight as UpdateIcon,
  PackageOpen as MyThings,
  Link2 as ChainsIcon,
  CircleUserRound as ProfileIcon,
  type LucideIcon,
} from "lucide-react";
import { FetchProfile } from "../../Widgets/FetchProfile/FetchProfile";

type NavigationItem = {
  to: string;
  label: string;
  Icon: LucideIcon;
};

const navigationItems: NavigationItem[] = [
  { to: "/trades", label: "Обмены", Icon: UpdateIcon },
  { to: "/my-items", label: "Мои вещи", Icon: MyThings },
  { to: "/my-chains", label: "Мои цепочки", Icon: ChainsIcon },
  // { to: "/profile", label: "Профиль", Icon: ProfileIcon },
];

const HeaderComponent = () => {
  return (
    <header className={styles.header}>
      <div className={styles.header__wrapp}>
        <NavLink
          to="/"
          className={styles.header__logoLink}
          aria-label="На главную"
        >
          <AvitoLogo className={styles.header__logo} />
        </NavLink>

        <nav className={styles.header__nav} aria-label="Основная навигация">
          <ul className={styles.header__navList}>
            {navigationItems.map(({ to, label, Icon }, id) => (
              <li key={id}>
                <NavLink
                  to={to}
                  className={({ isActive }) =>
                    `${styles.header__navLink} ${
                      isActive ? styles.header__navLink_active : ""
                    }`
                  }
                >
                  <Icon
                    className={styles.header__navIcon}
                    strokeWidth={1.8}
                    aria-hidden="true"
                  />

                  <span>{label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>

        <div className={styles.header__actions}>
          <Button size="m" color="green">
            Добавить вещь
          </Button>

          <FetchProfile/>
        </div>
      </div>
    </header>
  );
};

export const Header = memo(HeaderComponent);
