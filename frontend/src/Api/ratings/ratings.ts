import axios from "axios";

import api from "../client";

import {
  ExchangeRatingSchema,
  RateExchangeRequestSchema,
  RatingsPageSchema,
  type TExchangeRating,
  type TRatingsPage,
} from "./ratings.types";

// Кого оценивают, определяет сервер по самой цепочке, поэтому в теле только балл и текст.
export const rateExchangePartner = async (
  exchangeId: string,
  score: number,
  comment: string,
): Promise<TExchangeRating> => {
  const payload = RateExchangeRequestSchema.parse({ score, comment });
  const { data } = await api.put(`/exchanges/${exchangeId}/rating`, payload);

  return ExchangeRatingSchema.parse(data);
};

export const getUserRatings = async (
  userId: string,
  limit: number,
  offset: number,
): Promise<TRatingsPage> => {
  const { data } = await api.get(`/users/${userId}/ratings`, {
    params: { limit, offset },
  });

  return RatingsPageSchema.parse(data);
};

export const getRatingError = (error: unknown, fallback: string): string => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? fallback;
  }

  return fallback;
};
