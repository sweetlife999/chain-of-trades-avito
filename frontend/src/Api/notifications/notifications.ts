import api from "../client";

import {
  MarkAllNotificationsSchema,
  NotificationsPageSchema,
  type TNotificationsPage,
} from "./notifications.types";

type TGetNotificationsParams = {
  unreadOnly?: boolean;
  limit?: number;
  offset?: number;
};

export const getNotifications = async ({
  unreadOnly = false,
  limit = 30,
  offset = 0,
}: TGetNotificationsParams = {}): Promise<TNotificationsPage> => {
  const { data } = await api.get("/notifications", {
    params: {
      unread_only: unreadOnly,
      limit,
      offset,
    },
  });

  return NotificationsPageSchema.parse(data);
};

export const markNotificationRead = async (id: string): Promise<void> => {
  await api.post(`/notifications/${id}/read`);
};

export const markAllNotificationsRead = async (): Promise<number> => {
  const { data } = await api.post("/notifications/read-all");
  return MarkAllNotificationsSchema.parse(data).marked_count;
};
