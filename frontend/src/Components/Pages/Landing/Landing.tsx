import { memo, useEffect, useRef, type ReactNode } from "react";
import { Link, useNavigate } from "react-router-dom";

import styles from "./Styles.module.scss";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";

const steps: [string, string, string][] = [
  ["01", "Добавь вещь", "Опиши, что отдаёшь и что хочешь получить взамен."],
  [
    "02",
    "Сервис ищет цепочку",
    "Алгоритм находит замкнутую цепочку из нескольких участников.",
  ],
  ["03", "Все подтверждают", "Каждый решает — участвовать или отказаться."],
  ["04", "Относите вещь в пункт", "Каждый сдаёт свою вещь в пункт выдачи."],
  ["05", "Ожидание доставки", "Вещи едут между пунктами к новым владельцам."],
  ["06", "Забираете свою вещь", "Приходите в пункт и забираете то, что искали."],
];

const iconProps = {
  className: styles.benefitIcon,
  viewBox: "0 0 24 24",
  width: 28,
  height: 28,
  fill: "none",
  stroke: "var(--color-accent)",
  strokeWidth: 2,
  strokeLinecap: "round",
  strokeLinejoin: "round",
} as const;

const benefits: { icon: ReactNode; title: string; text: string }[] = [
  {
    icon: (
      <svg {...iconProps} aria-hidden="true">
        <circle cx="12" cy="12" r="9" />
        <line x1="5.5" y1="5.5" x2="18.5" y2="18.5" />
      </svg>
    ),
    title: "Без денег",
    text: "Обмен вещами напрямую — никаких доплат и комиссий.",
  },
  {
    icon: (
      <svg {...iconProps} aria-hidden="true">
        <line x1="6" y1="3" x2="6" y2="15" />
        <circle cx="18" cy="6" r="3" />
        <circle cx="6" cy="18" r="3" />
        <path d="M18 9a9 9 0 0 1-9 9" />
      </svg>
    ),
    title: "Больше совпадений",
    text: "Цепочка из нескольких участников находит совпадения, невозможные при обмене один на один.",
  },
  {
    icon: (
      <svg {...iconProps} aria-hidden="true">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z" />
        <path d="M9 12l2 2 4-4" />
      </svg>
    ),
    title: "Безопасность",
    text: "Блокировки и жалобы на участников защищают от недобросовестных обменов.",
  },
  {
    icon: (
      <svg {...iconProps} aria-hidden="true">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        <line x1="7" y1="8" x2="17" y2="8" />
        <line x1="7" y1="12" x2="13" y2="12" />
      </svg>
    ),
    title: "Прозрачность",
    text: "Весь тред обмена и решения участников видны на каждом этапе.",
  },
];

const faq: [string, string][] = [
  ["Это бесплатно?", "Да. Сервис не берёт денег за обмен — только вещи на вещи."],
  [
    "Что если кто-то из цепочки откажется?",
    "Обмен не состоится для всех участников, сервис продолжит искать цепочку заново.",
  ],
  [
    "Как происходит передача вещи?",
    "Договариваетесь с участником напрямую или используете пункт выдачи.",
  ],
  [
    "Что делать, если меня обманули?",
    "Оставьте жалобу в треде обмена — она уходит в очередь модерации.",
  ],
];

const LandingComponent = () => {
  const navigate = useNavigate();
  const { isAuth } = useAuthSelector();
  const rootRef = useRef<HTMLDivElement>(null);
  const navRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const root = rootRef.current;
    const nav = navRef.current;

    if (!root || !nav) {
      return;
    }

    const navSize = new ResizeObserver(() =>
      root.style.setProperty("--nav-h", `${nav.offsetHeight}px`),
    );
    navSize.observe(nav);

    const reveal = new IntersectionObserver(
      (entries) =>
        entries.forEach((entry) => {
          if (!entry.isIntersecting) {
            return;
          }

          entry.target.classList.add(styles.visible);
          reveal.unobserve(entry.target);
        }),
      { rootMargin: "0px 0px -10% 0px" },
    );
    root
      .querySelectorAll(`.${styles.section}`)
      .forEach((section) => reveal.observe(section));

    return () => {
      navSize.disconnect();
      reveal.disconnect();
    };
  }, []);

  const startExchange = () =>
    isAuth
      ? navigate("/exchanges/create")
      : navigate("/login", { state: { from: "/exchanges/create" } });

  return (
    <div className={styles.landing} ref={rootRef}>
      <nav className={styles.nav} ref={navRef} aria-label="Навигация лендинга">
        <span className={styles.navBrand}>Цепочка обмена</span>

        <div className={styles.navLinks}>
          <a className={styles.navLink} href="#how">
            Как это работает
          </a>
          <a className={styles.navLink} href="#benefits">
            Преимущества
          </a>
          <a className={styles.navLink} href="#faq">
            FAQ
          </a>
          <Link
            className={`${styles.btn} ${styles.btnPrimary}`}
            to="/login"
            state={{ from: "/feed", closeTo: "/" }}
          >
            Войти
          </Link>
        </div>
      </nav>

      <div className={styles.container}>
        <section className={`${styles.section} ${styles.hero}`}>
          <h1 className={styles.heroTitle}>
            <span className={`${styles.heroLine} ${styles.heroReveal}`}>
              Отдай <span className={styles.heroGive}>ненужное</span>.
            </span>
            <span className={`${styles.heroLine} ${styles.heroReveal}`}>
              Получи <span className={styles.heroGet}>нужное</span>.
            </span>
          </h1>
          <p className={`${styles.heroText} ${styles.heroReveal}`}>
            Цепочка обмена собирает нескольких человек в замкнутую цепочку: каждый
            отдаёт вещь, которая ему не нужна, и получает ту, что искал. Без денег
            и посредников — просто цепочка людей, которым это выгодно всем сразу.
          </p>
        </section>

        <section className={styles.section} id="how">
          <span className={styles.kicker}>Как это работает</span>

          <div className={styles.steps}>
            {steps.map(([number, title, text]) => (
              <div className={styles.step} key={number}>
                <span className={styles.stepNumber}>{number}</span>
                <h2 className={styles.stepTitle}>{title}</h2>
                <p className={styles.stepText}>{text}</p>
              </div>
            ))}
          </div>
        </section>

        <section className={styles.section} id="benefits">
          <span className={styles.kicker}>Преимущества обмена цепочкой</span>

          <div className={styles.benefits}>
            {benefits.map(({ icon, title, text }) => (
              <div key={title}>
                {icon}
                <h3 className={styles.benefitTitle}>{title}</h3>
                <p className={styles.benefitText}>{text}</p>
              </div>
            ))}
          </div>
        </section>

        <section className={styles.section} id="faq">
          <span className={styles.kicker}>FAQ</span>

          <div className={styles.faq}>
            {faq.map(([question, answer]) => (
              <div className={styles.faqItem} key={question}>
                <h3 className={styles.faqTitle}>{question}</h3>
                <p className={styles.faqText}>{answer}</p>
              </div>
            ))}
          </div>
        </section>
      </div>

      <section className={`${styles.section} ${styles.cta}`}>
        <div className={styles.ctaInner}>
          <h3 className={styles.ctaTitle}>Готовы обменяться?</h3>
          <div className={styles.ctaActions}>
            <button
              className={`${styles.btn} ${styles.btnInverse}`}
              type="button"
              onClick={startExchange}
            >
              Начать обмен
            </button>
          </div>
        </div>
      </section>
    </div>
  );
};

export const Landing = memo(LandingComponent);
