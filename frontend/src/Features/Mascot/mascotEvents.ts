import type { MascotEvent, MascotReaction } from "./mascot.types";

export const mascotReactions: Record<MascotEvent, MascotReaction> = {
  CHAT_OPENED: {
    mood: "idle",
    mode: "dialog",
    message: "Я рядом. Могу подсказать, с чего начать разговор.",
    durationMs: 4200,
  },
  USER_TYPING: {
    mood: "listening",
    mode: "ambient",
    message: null,
    durationMs: 1800,
  },
  HINT_SHOWN: {
    mood: "hint",
    mode: "attention",
    message: "Не знаете, что написать? Выберите подсказку — текст можно изменить перед отправкой.",
    durationMs: 5200,
  },
  HINT_SELECTED: {
    mood: "happy",
    mode: "dialog",
    message: "Отлично! Я вставил текст — проверьте его и отправляйте.",
    durationMs: 3200,
  },
  MESSAGE_SENT: {
    mood: "thinking",
    mode: "attention",
    message: "Отправляю…",
    durationMs: 2600,
  },
  MESSAGE_SENT_SUCCESS: {
    mood: "happy",
    mode: "attention",
    message: "Сообщение отправлено ✓",
    durationMs: 2200,
  },
  MESSAGE_RECEIVED: {
    mood: "happy",
    mode: "attention",
    message: "Есть новый ответ ✨",
    durationMs: 2800,
  },
  PARTICIPANT_ACCEPTED: {
    mood: "happy",
    mode: "attention",
    message: "Ещё один участник подтвердил обмен.",
    durationMs: 3200,
  },
  PARTICIPANT_DECLINED: {
    mood: "concerned",
    mode: "dialog",
    message: "Кто-то отказался. Ничего страшного — сервис сможет искать новую цепочку.",
    durationMs: 5200,
  },
  PARTICIPANT_DELIVERED: {
    mood: "happy",
    mode: "attention",
    message: "Вещь уже в пункте выдачи. Продвигаемся!",
    durationMs: 3600,
  },
  EXCHANGE_CONFIRMED: {
    mood: "celebrate",
    mode: "dialog",
    message: "Все подтвердили обмен! 🎉 Теперь можно переходить к передаче вещей.",
    durationMs: 5200,
  },
  EXCHANGE_DELIVERING: {
    mood: "happy",
    mode: "attention",
    message: "Все вещи приняты — доставка началась.",
    durationMs: 4200,
  },
  EXCHANGE_DELIVERED: {
    mood: "celebrate",
    mode: "dialog",
    message: "Вещи приехали! Осталось забрать свою в пункте выдачи.",
    durationMs: 5000,
  },
  EXCHANGE_COMPLETED: {
    mood: "celebrate",
    mode: "dialog",
    message: "Готово! Цепочка обмена успешно завершена 🎉",
    durationMs: 6000,
  },
  ERROR: {
    mood: "concerned",
    mode: "dialog",
    message: "Похоже, что-то пошло не так. Попробуйте ещё раз.",
    durationMs: 4200,
  },
};
