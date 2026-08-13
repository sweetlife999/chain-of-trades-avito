import { describe, expect, it } from "vitest";

import type { TNotification } from "../../../Api/notifications/notifications.types";
import {
  getNotificationDescription,
  getNotificationPath,
  getNotificationTitle,
} from "./notificationPresentation";

const createNotification = (
  overrides: Partial<TNotification> = {},
): TNotification => ({
  actor: { id: "user-2", nickname: "Анна", photo_url: null },
  created_at: "2026-08-13T10:00:00.000Z",
  exchange_id: "exchange-1",
  exchange_status: "proposed",
  gives_item_title: "Телефон",
  id: "notification-1",
  is_read: false,
  kind: "text",
  message_id: "message-1",
  read_at: null,
  receives_item_title: "Книга",
  support_subject: "",
  support_thread_id: null,
  target_type: "exchange",
  ...overrides,
});

describe("notification presentation", () => {
  it("shows the message author and exchange description", () => {
    const notification = createNotification();

    expect(getNotificationTitle(notification)).toBe("Сообщение от Анна");
    expect(getNotificationDescription(notification)).toBe(
      "Телефон → Книга. Откройте цепочку, чтобы прочитать сообщение.",
    );
    expect(getNotificationPath(notification, false)).toBe(
      "/exchanges/exchange-1",
    );
  });

  it("routes a user support notification to the support chat", () => {
    const notification = createNotification({
      exchange_id: null,
      kind: "support_message",
      support_subject: "Проблема с обменом",
      support_thread_id: "thread-1",
      target_type: "support",
    });

    expect(getNotificationTitle(notification)).toBe("Поддержка: Анна");
    expect(getNotificationDescription(notification)).toBe(
      "Проблема с обменом",
    );
    expect(getNotificationPath(notification, false)).toBe(
      "/support?thread=thread-1",
    );
  });

  it("routes an administrator to the support section", () => {
    const notification = createNotification({
      exchange_id: null,
      kind: "support_message",
      support_thread_id: "thread-1",
      target_type: "support",
    });

    expect(getNotificationPath(notification, true)).toBe(
      "/admin?section=support&thread=thread-1",
    );
  });

  it("falls back to notifications when no target id is available", () => {
    const notification = createNotification({ exchange_id: null });

    expect(getNotificationPath(notification, false)).toBe("/notifications");
  });
});

