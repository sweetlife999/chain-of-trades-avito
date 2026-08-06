import { z } from "zod";

export const ExchangeStatusSchema = z.enum([
  "proposed",
  "confirmed",
  "completed",
  "cancelled",
]);

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
  decided_at: z.string().nullable().optional(),
  gives_item: ExchangeItemSchema,
  position: z.number(),
  receives_item: ExchangeItemSchema,
  status: z.string(),
  user: z.object({
    id: z.string(),
    nickname: z.string(),
    photo_url: z.string().nullable().optional(),
  }),
});

export const ExchangeSchema = z.object({
  closed_at: z.string().nullable().optional(),
  created_at: z.string(),
  id: z.string(),
  participants: z.array(ExchangeParticipantSchema),
  status: ExchangeStatusSchema,
  updated_at: z.string(),
});

export const ExchangesSchema = z.array(ExchangeSchema);

export type TExchangeStatus = z.infer<typeof ExchangeStatusSchema>;
export type TExchange = z.infer<typeof ExchangeSchema>;
export type TExchangeParticipant = z.infer<typeof ExchangeParticipantSchema>;
export type TExchanges = z.infer<typeof ExchangesSchema>;
