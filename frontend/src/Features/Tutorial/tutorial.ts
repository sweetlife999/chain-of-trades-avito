export type TutorialTarget =
  | "exchanges"
  | "my-items"
  | "my-chains"
  | "support"
  | "add-item"
  | "notifications"
  | "profile";

export type TutorialStep = {
  message: string;
  target: TutorialTarget;
};

export const tutorialSteps: readonly TutorialStep[] = [
  {
    target: "exchanges",
    message:
      "Здесь находятся доступные обмены. Я помогу заметить новые подходящие цепочки.",
  },
  {
    target: "my-items",
    message:
      "В разделе «Мои вещи» можно посмотреть свои объявления и управлять ими.",
  },
  {
    target: "my-chains",
    message:
      "В «Моих цепочках» видно состояние каждого вашего обмена.",
  },
  {
    target: "support",
    message:
      "Если появится вопрос, здесь можно открыть обычный чат с поддержкой.",
  },
  {
    target: "add-item",
    message:
      "С этой кнопки начинается обмен: добавьте вещь и укажите, что хотите получить.",
  },
  {
    target: "notifications",
    message:
      "Колокольчик сообщит о новых цепочках, сообщениях и этапах обмена.",
  },
  {
    target: "profile",
    message:
      "Здесь находится ваш профиль, отзывы и настройки аккаунта. Знакомство завершено!",
  },
] as const;

const storagePrefix = "umi-onboarding";

const getStorageKey = (userId: string) => `${storagePrefix}:${userId}`;

export const saveTutorialProgress = (userId: string, step: number) => {
  try {
    window.localStorage.setItem(getStorageKey(userId), `active:${step}`);
  } catch {
    // Туториал продолжит работать в текущей сессии без localStorage.
  }
};

export const readTutorialProgress = (userId: string): number | null => {
  try {
    const value = window.localStorage.getItem(getStorageKey(userId));
    if (!value?.startsWith("active:")) {
      return null;
    }

    const step = Number(value.slice("active:".length));
    return Number.isInteger(step) && step >= 0 && step < tutorialSteps.length
      ? step
      : 0;
  } catch {
    return null;
  }
};

export const completeTutorial = (userId: string) => {
  try {
    window.localStorage.setItem(getStorageKey(userId), "completed");
  } catch {
    // Завершение всё равно сохранится в Redux до конца текущей сессии.
  }
};

export const isHeaderNavigationTutorialTarget = (
  target: TutorialTarget | undefined,
) =>
  target === "exchanges" ||
  target === "my-items" ||
  target === "my-chains" ||
  target === "support" ||
  target === "add-item";
