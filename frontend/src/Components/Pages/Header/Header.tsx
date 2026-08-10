import { memo, useEffect, useState } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import clsx from "clsx";
import {
  ArrowLeftRight as UpdateIcon,
  PackageOpen as MyThings,
  Link2 as ChainsIcon,
  Menu as MenuIcon,
  ShieldCheck,
  UserRound as ProfileIcon,
  X as CloseMenuIcon,
  type LucideIcon,
} from "lucide-react";

import AvitoLogo from "/src/Assets/avito-logo.svg";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { FetchProfile } from "../../Widgets/FetchProfile/FetchProfile";
import { Notifications } from "../../Widgets/Notifications/Notifications";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { useLogout } from "../../../Hooks/useLogout";

// menuOnly — пункт живёт только в бургер-панели, в строке шапки его нет.
type NavigationItem = {
  to: string;
  label: string;
  Icon: LucideIcon;
  menuOnly?: boolean;
};

const navigationItems: NavigationItem[] = [
  { to: "/feed", label: "Обмены", Icon: UpdateIcon },
  { to: "/myItems", label: "Мои вещи", Icon: MyThings },
  { to: "/exchanges", label: "Мои цепочки", Icon: ChainsIcon },
];

const adminNavigationItem: NavigationItem = {
  to: "/admin",
  label: "Админка",
  Icon: ShieldCheck,
};

const profileNavigationItem: NavigationItem = {
  to: "/profile",
  label: "Профиль",
  Icon: ProfileIcon,
  menuOnly: true,
};

const navLinkClassName = ({ isActive }: { isActive: boolean }) =>
  clsx(styles.header__navLink, isActive && styles.header__navLink_active);

const menuOnlyItemClassName = clsx(
  styles.header__navItem,
  styles.header__navItem_menuOnly,
);

const HeaderComponent = () => {
  const navigate = useNavigate();
  const { isAdmin, isAuth } = useAuthSelector();
  const handleLogout = useLogout();
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  const visibleNavigationItems = [
    ...navigationItems,
    ...(isAdmin ? [adminNavigationItem] : []),
    ...(isAuth ? [profileNavigationItem] : []),
  ];

  useEffect(() => {
    if (!isMenuOpen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMenuOpen(false);
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isMenuOpen]);

  const goToItemCreation = () =>
    isAuth
      ? navigate("/exchanges/create")
      : navigate("/login", { state: { from: "/exchanges/create" } });

  const BurgerIcon = isMenuOpen ? CloseMenuIcon : MenuIcon;

  return (
    <header className={styles.header}>
      <div className={styles.header__wrapper}>
        <NavLink to="/feed" className={styles.header__logoLink} aria-label="На главную">
          <img className={styles.header__logo} src={AvitoLogo} alt="Авито Обмен" />
        </NavLink>

        <nav
          aria-label="Основная навигация"
          className={clsx(styles.header__nav, isMenuOpen && styles.header__nav_open)}
          id="header-menu"
        >
          {/* Один обработчик на список: закрывает панель и по ссылке, и по «Добавить вещь». */}
          <ul className={styles.header__navList} onClick={() => setIsMenuOpen(false)}>
            {visibleNavigationItems.map(({ to, label, Icon, menuOnly }) => (
              <li
                className={menuOnly ? menuOnlyItemClassName : styles.header__navItem}
                key={to}
              >
                <NavLink to={to} className={navLinkClassName}>
                  <Icon className={styles.header__navIcon} strokeWidth={1.8} aria-hidden="true" />
                  {label}
                </NavLink>
              </li>
            ))}

            <li className={menuOnlyItemClassName}>
              <Button
                className={styles.header__menuButton}
                color="green"
                onClick={goToItemCreation}
              >
                Добавить вещь
              </Button>
            </li>

            {isAuth && (
              <li className={menuOnlyItemClassName}>
                <Button
                  className={styles.header__menuButton}
                  color="transparent"
                  onClick={handleLogout}
                >
                  Выйти
                </Button>
              </li>
            )}
          </ul>
        </nav>

        <div className={styles.header__actions}>
          <Button
            className={styles.header__addButton}
            size="m"
            color="green"
            onClick={goToItemCreation}
          >
            Добавить вещь
          </Button>
          <Notifications />
          <FetchProfile />

          <button
            aria-controls="header-menu"
            aria-expanded={isMenuOpen}
            aria-label="Меню"
            className={styles.header__burger}
            type="button"
            onClick={() => setIsMenuOpen((open) => !open)}
          >
            <BurgerIcon aria-hidden="true" size={22} strokeWidth={1.9} />
          </button>
        </div>
      </div>
    </header>
  );
};

export const Header = memo(HeaderComponent);
