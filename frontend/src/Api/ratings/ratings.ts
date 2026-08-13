import axios from "axios";

import api from "../client";

import {
  ExchangeRatingSchema,
  RateExchangeRequestSchema,
  RatingsPageSchema,
  type TExchangeRating,
  type TRateExchangeRequest,
  type TRatingsPage,
} from "./ratings.types";

// Кого оценивают, определяет сервер по самой цепочке, поэтому ID пользователя
// в запрос не передаём — только балл и необязательный комментарий.
export const rateExchangePartner = async (
  exchangeId: string,
  request: TRateExchangeRequest,
): Promise<TExchangeRating> => {
  const payload = RateExchangeRequestSchema.parse(request);
  const { data } = await api.put(`/exchanges/${exchangeId}/rating`, payload);

  return ExchangeRatingSchema.parse(data);
};

export const getUserRatings = async (
  userId: string,
  limit = 20,
  offset = 0,
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
