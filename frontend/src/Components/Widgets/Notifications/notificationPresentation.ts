import type {
  TNotification,
  TNotificationKind,
} from "../../../Api/notifications/notifications.types";

const kindTitles: Record<TNotificationKind, string> = {
  exchange_proposed: "Найдена новая цепочка",
  text: "Новое сообщение",
  participant_accepted: "Участник подтвердил обмен",
  participant_declined: "Участник отказался от обмена",
  participant_completed: "Участник подтвердил получение",
  participant_delivered_item: "Товар передан в ПВЗ",
  exchange_confirmed: "Обмен подтверждён",
  exchange_delivering: "Вещи отправлены между ПВЗ",
  exchange_delivered: "Вещь доставлена в ПВЗ",
  exchange_completed: "Обмен завершён",
  exchange_superseded: "Предложение обмена закрыто",
  exchange_item_withdrawn: "Объявление снято с поиска",
};

export const getNotificationTitle = (notification: TNotification) => {
  if (notification.kind === "text" && notification.actor) {
    return `Сообщение от ${notification.actor.nickname}`;
  }
  if (
    notification.actor &&
    notification.kind.startsWith("participant_")
  ) {
    return `${kindTitles[notification.kind]}: ${notification.actor.nickname}`;
  }
  return kindTitles[notification.kind];
};

export const getNotificationDescription = (notification: TNotification) => {
  const exchange = `${notification.gives_item_title} → ${notification.receives_item_title}`;

  if (notification.kind === "text") {
    return `${exchange}. Откройте цепочку, чтобы прочитать сообщение.`;
  }
  return exchange;
};

export const formatNotificationTime = (value: string) => {
  const date = new Date(value);
  const now = new Date();
  const sameDay = date.toDateString() === now.toDateString();

  if (sameDay) {
    return `Сегодня, ${new Intl.DateTimeFormat("ru-RU", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date)}`;
  }

  return new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "long",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
};
