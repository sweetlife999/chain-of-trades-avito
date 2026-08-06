import { z } from "zod";

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
  decided_at: z.string(),//.nullable()
  gives_item: ExchangeItemSchema,
  position: z.number(),
  receives_item: ExchangeItemSchema,
  status: z.string(),
  user: z.object({
    id: z.string(),
    nickname: z.string(),
    photo_url: z.string(),//.nullable()
  }),
});

export const ExchangeSchema = z.object({
  closed_at: z.string(),//.nullable()
  created_at: z.string(),
  id: z.string(),
  participants: z.array(ExchangeParticipantSchema),
  status: z.string(),
  updated_at: z.string(),
});

export const ExchangesSchema = z.array(ExchangeSchema);

export type TExchange = z.infer<typeof ExchangeSchema>;
export type TExchanges = z.infer<typeof ExchangesSchema>;
