import { memo, useEffect, useState } from "react";
import { useSelector } from "react-redux";
import { NavLink, useNavigate } from "react-router-dom";
import clsx from "clsx";
import {
  ArrowLeftRight as UpdateIcon,
  PackageOpen as MyThings,
  Link2 as ChainsIcon,
  Menu as MenuIcon,
  ShieldCheck,
  Headset,
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
import {
  isHeaderNavigationTutorialTarget,
  tutorialSteps,
  type TutorialTarget,
} from "../../../Features/Tutorial/tutorial";
import type { RootState } from "../../../Store/store";

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
  { to: "/support", label: "Поддержка", Icon: Headset },
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

const tutorialTargetByPath: Partial<Record<string, TutorialTarget>> = {
  "/feed": "exchanges",
  "/myItems": "my-items",
  "/exchanges": "my-chains",
  "/support": "support",
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
  const tutorial = useSelector((state: RootState) => state.tutorial);
  const tutorialTarget = tutorial.active
    ? tutorialSteps[tutorial.step]?.target
    : undefined;
  const menuOpen =
    isMenuOpen || isHeaderNavigationTutorialTarget(tutorialTarget);

  const visibleNavigationItems = [
    ...navigationItems,
    ...(isAdmin ? [adminNavigationItem] : []),
    ...(isAuth ? [profileNavigationItem] : []),
  ];

  useEffect(() => {
    if (!menuOpen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMenuOpen(false);
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [menuOpen]);

  const goToItemCreation = () =>
    isAuth
      ? navigate("/exchanges/create")
      : navigate("/login", { state: { from: "/exchanges/create" } });

  const BurgerIcon = menuOpen ? CloseMenuIcon : MenuIcon;

  return (
    <header className={styles.header}>
      <div className={styles.header__wrapper}>
        <NavLink to="/feed" className={styles.header__logoLink} aria-label="На главную">
          <img className={styles.header__logo} src={AvitoLogo} alt="Авито Обмен" />
        </NavLink>

        <nav
          aria-label="Основная навигация"
          className={clsx(styles.header__nav, menuOpen && styles.header__nav_open)}
          id="header-menu"
        >
          {/* Один обработчик на список: закрывает панель и по ссылке, и по «Добавить вещь». */}
          <ul className={styles.header__navList} onClick={() => setIsMenuOpen(false)}>
            {visibleNavigationItems.map(({ to, label, Icon, menuOnly }) => (
              <li
                className={menuOnly ? menuOnlyItemClassName : styles.header__navItem}
                key={to}
              >
                <NavLink
                  className={navLinkClassName}
                  data-tutorial-target={tutorialTargetByPath[to]}
                  to={to}
                >
                  <Icon className={styles.header__navIcon} strokeWidth={1.8} aria-hidden="true" />
                  {label}
                </NavLink>
              </li>
            ))}

            <li className={menuOnlyItemClassName}>
              <Button
                className={styles.header__menuButton}
                color="green"
                tutorialTarget="add-item"
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
            tutorialTarget="add-item"
            onClick={goToItemCreation}
          >
            Добавить вещь
          </Button>
          <Notifications />
          <FetchProfile />

          <button
            aria-controls="header-menu"
            aria-expanded={menuOpen}
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
