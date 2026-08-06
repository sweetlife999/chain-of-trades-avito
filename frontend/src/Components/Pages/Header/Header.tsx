import { memo } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import {
  ArrowLeftRight as UpdateIcon,
  PackageOpen as MyThings,
  Link2 as ChainsIcon,
  type LucideIcon,
} from "lucide-react";

import AvitoLogo from "/src/Assets/avito-logo.svg?react";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { FetchProfile } from "../../Widgets/FetchProfile/FetchProfile";

type NavigationItem = { to: string; label: string; Icon: LucideIcon };

const navigationItems: NavigationItem[] = [
  { to: "/", label: "Обмены", Icon: UpdateIcon },
  { to: "/myItems", label: "Мои вещи", Icon: MyThings },
  { to: "/myChains", label: "Мои цепочки", Icon: ChainsIcon },
];

const HeaderComponent = () => {
  const navigate = useNavigate();

  return (
    <header className={styles.header}>
      <div className={styles.header__wrapp}>
        <NavLink to="/" className={styles.header__logoLink} aria-label="На главную">
          <AvitoLogo className={styles.header__logo} />
        </NavLink>

        <nav className={styles.header__nav} aria-label="Основная навигация">
          <ul className={styles.header__navList}>
            {navigationItems.map(({ to, label, Icon }) => (
              <li key={to}>
                <NavLink
                  to={to}
                  end={to === "/"}
                  className={({ isActive }) =>
                    `${styles.header__navLink} ${isActive ? styles.header__navLink_active : ""}`
                  }
                >
                  <Icon className={styles.header__navIcon} strokeWidth={1.8} aria-hidden="true" />
                  <span>{label}</span>
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>

        <div className={styles.header__actions}>
          <Button size="m" color="green" onClick={() => navigate("/exchanges/create")}>
            Добавить вещь
          </Button>
          <FetchProfile />
        </div>
      </div>
    </header>
  );
};

export const Header = memo(HeaderComponent);
