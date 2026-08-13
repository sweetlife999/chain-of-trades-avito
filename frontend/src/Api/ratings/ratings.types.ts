import { z } from "zod";

export const RateExchangeRequestSchema = z.object({
  score: z.number().int().min(1).max(5),
  // По Swagger комментарий необязателен. Не вводим клиентский лимит, которого
  // нет в контракте API.
  comment: z.string().trim().optional(),
});

export const ExchangeRatingSchema = z.object({
  rated_user_id: z.string(),
  score: z.number().int().min(1).max(5),
  comment: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

// Автора в ленте нет: Swagger намеренно возвращает отзывы анонимно.
export const ReceivedRatingSchema = z.object({
  score: z.number().int().min(1).max(5),
  comment: z.string(),
  created_at: z.string(),
});

export const RatingsPageSchema = z.object({
  ratings: z.array(ReceivedRatingSchema),
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().nonnegative(),
});

export type TRateExchangeRequest = z.infer<typeof RateExchangeRequestSchema>;
export type TExchangeRating = z.infer<typeof ExchangeRatingSchema>;
export type TReceivedRating = z.infer<typeof ReceivedRatingSchema>;
export type TRatingsPage = z.infer<typeof RatingsPageSchema>;
