import { z } from "zod";

export const NotificationKindSchema = z.enum([
  "exchange_proposed",
  "text",
  "participant_accepted",
  "participant_declined",
  "participant_completed",
  "participant_delivered_item",
  "exchange_confirmed",
  "exchange_delivering",
  "exchange_delivered",
  "exchange_completed",
  "exchange_superseded",
  "exchange_item_withdrawn",
]);

export const NotificationActorSchema = z.object({
  id: z.string(),
  nickname: z.string(),
  photo_url: z.string().nullable().optional(),
});

export const NotificationSchema = z.object({
  id: z.string(),
  kind: NotificationKindSchema,
  exchange_id: z.string(),
  message_id: z.string().nullable(),
  actor: NotificationActorSchema.nullable(),
  exchange_status: z.string(),
  gives_item_title: z.string(),
  receives_item_title: z.string(),
  is_read: z.boolean(),
  read_at: z.string().nullable(),
  created_at: z.string(),
});

export const NotificationsPageSchema = z.object({
  notifications: z.array(NotificationSchema),
  unread_count: z.number().int().nonnegative(),
  limit: z.number().int().positive(),
  offset: z.number().int().nonnegative(),
});

export const MarkAllNotificationsSchema = z.object({
  marked_count: z.number().int().nonnegative(),
});

export type TNotificationKind = z.infer<typeof NotificationKindSchema>;
export type TNotification = z.infer<typeof NotificationSchema>;
export type TNotificationsPage = z.infer<typeof NotificationsPageSchema>;
