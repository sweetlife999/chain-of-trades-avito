import { z } from "zod";

import { OptionalPhotoUrlResponseSchema } from "../common.types";

export const SupportPersonSchema = z.object({
  id: z.string(),
  nickname: z.string(),
  photo_url: OptionalPhotoUrlResponseSchema,
  is_admin: z.boolean(),
});

export const SupportThreadSchema = z.object({
  id: z.string(),
  user: SupportPersonSchema.nullish(),
  subject: z.string(),
  status: z.enum(["open", "in_progress", "closed"]),
  assigned_admin: SupportPersonSchema.nullable(),
  last_message: z.string().optional().default(""),
  last_message_at: z.string().nullable(),
  unread_count: z.number().int().nonnegative(),
  created_at: z.string(),
  updated_at: z.string(),
  closed_at: z.string().nullable(),
});

export const SupportMessageSchema = z.object({
  id: z.string(),
  thread_id: z.string(),
  author: SupportPersonSchema,
  body: z.string(),
  created_at: z.string(),
});

export const SupportThreadMessagesSchema = z.object({
  thread: SupportThreadSchema,
  messages: z.array(SupportMessageSchema),
});

export const AdminSupportPageSchema = z.object({
  threads: z.array(SupportThreadSchema),
  total: z.number().int().nonnegative(),
  limit: z.number().int().positive(),
  offset: z.number().int().nonnegative(),
});

export type TSupportThread = z.infer<typeof SupportThreadSchema>;
export type TSupportMessage = z.infer<typeof SupportMessageSchema>;
export type TSupportThreadMessages = z.infer<
  typeof SupportThreadMessagesSchema
>;
export type TAdminSupportPage = z.infer<typeof AdminSupportPageSchema>;
