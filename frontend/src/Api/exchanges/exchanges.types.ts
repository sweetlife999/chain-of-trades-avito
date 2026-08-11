import { z } from "zod";

export const ExchangeStatusSchema = z.enum([
  "proposed",
  "confirmed",
  "delivering",
  "delivered",
  "completed",
  "cancelled",
]);

export const ExchangeUserSchema = z.object({
  id: z.string(),
  nickname: z.string(),
  photo_url: z
    .url("Введите корректную ссылку")
    .regex(/^https?:\/\//, "Должна быть ссылка с http или https")
    .nullable()
    .optional(),
});

export const ExchangeItemSchema = z.object({
  category: z.object({
    name: z.string(),
    slug: z.string(),
  }),
  description: z.string(),
  id: z.string(),
  pickup_point: z
    .object({
      address: z.string(),
      id: z.string(),
      name: z.string(),
    })
    .nullable()
    .optional()
    .default(null),
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
  unread_count: z.number().int().nonnegative().default(0),
  updated_at: z.string(),
});

export const ExchangesSchema = z.array(ExchangeSchema);

export const CreateMessageRequestSchema = z.object({
  body: z.string().trim().min(1).max(2000),
});

export const MarkReadRequestSchema = z.object({
  last_message_id: z.string().min(1),
});

export const ExchangeMessageKindSchema = z.enum([
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
]);

export const ExchangeMessageSchema = z.object({
  author: ExchangeUserSchema.nullable().optional(),
  body: z.string().nullable(),
  created_at: z.string(),
  id: z.string(),
  kind: ExchangeMessageKindSchema,
});

export const ExchangeMessagesSchema = z.array(ExchangeMessageSchema);

export type TExchangeStatus = z.infer<typeof ExchangeStatusSchema>;
export type TExchange = z.infer<typeof ExchangeSchema>;
export type TExchangeParticipant = z.infer<typeof ExchangeParticipantSchema>;
export type TExchanges = z.infer<typeof ExchangesSchema>;
export type TCreateMessageRequest = z.infer<typeof CreateMessageRequestSchema>;
export type TMarkReadRequest = z.infer<typeof MarkReadRequestSchema>;
export type TExchangeMessage = z.infer<typeof ExchangeMessageSchema>;
export type TExchangeMessages = z.infer<typeof ExchangeMessagesSchema>;
