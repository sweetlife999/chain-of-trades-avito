import { z } from "zod";

export const ExchangeStatusSchema = z.enum([
  "proposed",
  "confirmed",
  "completed",
  "cancelled",
]);

export const ExchangeUserSchema = z.object({
  id: z.string(),
  nickname: z.string(),
  photo_url: z.string().nullable().optional(),
});

export const ExchangeItemSchema = z.object({
  category: z.object({
    name: z.string(),
    slug: z.string(),
  }),
  description: z.string(),
  id: z.string(),
  status: z.string(),
  title: z.string(),
});

export const ExchangeParticipantSchema = z.object({
  completion_confirmed_at: z.string().nullable().optional(),
  decided_at: z.string().nullable().optional(),
  gives_item: ExchangeItemSchema,
  position: z.number(),
  receives_item: ExchangeItemSchema,
  status: z.string(),
  user: ExchangeUserSchema,
});

export const ExchangeSchema = z.object({
  closed_at: z.string().nullable().optional(),
  created_at: z.string(),
  id: z.string(),
  participants: z.array(ExchangeParticipantSchema),
  status: ExchangeStatusSchema,
  unread_count: z.number().int().nonnegative().optional().default(0),
  updated_at: z.string(),
});

export const ExchangesSchema = z.array(ExchangeSchema);

export const CreateMessageRequestSchema = z.object({
  body: z.string().trim().min(1).max(2000),
});

export const ExchangeMessageSchema = z.object({
  author: ExchangeUserSchema.nullable().optional(),
  body: z.string(),
  created_at: z.string(),
  id: z.string(),
  kind: z.string(),
});

export const ExchangeMessagesSchema = z.array(ExchangeMessageSchema);

export type TExchangeStatus = z.infer<typeof ExchangeStatusSchema>;
export type TExchange = z.infer<typeof ExchangeSchema>;
export type TExchangeParticipant = z.infer<typeof ExchangeParticipantSchema>;
export type TExchanges = z.infer<typeof ExchangesSchema>;
export type TCreateMessageRequest = z.infer<typeof CreateMessageRequestSchema>;
export type TExchangeMessage = z.infer<typeof ExchangeMessageSchema>;
export type TExchangeMessages = z.infer<typeof ExchangeMessagesSchema>;
