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
  support_message: "Новое сообщение в поддержке",
};

export const getNotificationTitle = (notification: TNotification) => {
  if (notification.kind === "support_message" && notification.actor) {
    return `Поддержка: ${notification.actor.nickname}`;
  }
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
  if (notification.kind === "support_message") {
    return notification.support_subject || "Откройте чат поддержки, чтобы прочитать сообщение.";
  }

  const exchange = `${notification.gives_item_title} → ${notification.receives_item_title}`;

  if (notification.kind === "text") {
    return `${exchange}. Откройте цепочку, чтобы прочитать сообщение.`;
  }
  return exchange;
};

export const getNotificationPath = (
  notification: TNotification,
  isAdmin: boolean,
) => {
  if (notification.target_type === "support" && notification.support_thread_id) {
    return isAdmin
      ? `/admin?section=support&thread=${notification.support_thread_id}`
      : `/support?thread=${notification.support_thread_id}`;
  }

  return notification.exchange_id
    ? `/exchanges/${notification.exchange_id}`
    : "/notifications";
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
