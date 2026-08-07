import api from "../client";

import {
  CreateMessageRequestSchema,
  ExchangeMessageSchema,
  ExchangeMessagesSchema,
  ExchangeSchema,
  ExchangesSchema,
  MarkReadRequestSchema,
  type TExchange,
  type TExchangeMessage,
  type TExchangeMessages,
  type TExchanges,
} from "./exchanges.types";

export const getExchanges = async (): Promise<TExchanges> => {
  const { data } = await api.get("/exchanges");
  return ExchangesSchema.parse(data);
};

export const getExchange = async (id: string): Promise<TExchange> => {
  const { data } = await api.get(`/exchanges/${id}`);
  return ExchangeSchema.parse(data);
};

export const confirmExchange = async (id: string): Promise<void> => {
  await api.post(`/exchanges/${id}/confirm`);
};

export const declineExchange = async (id: string): Promise<void> => {
  await api.post(`/exchanges/${id}/decline`);
};

export const completeExchange = async (id: string): Promise<void> => {
  await api.post(`/exchanges/${id}/complete`);
};

export const getExchangeMessages = async (
  id: string,
): Promise<TExchangeMessages> => {
  const { data } = await api.get(`/exchanges/${id}/messages`);
  return ExchangeMessagesSchema.parse(data);
};

export const sendExchangeMessage = async (
  id: string,
  body: string,
): Promise<TExchangeMessage> => {
  const payload = CreateMessageRequestSchema.parse({ body });
  const { data } = await api.post(`/exchanges/${id}/messages`, payload);

  return ExchangeMessageSchema.parse(data);
};

export const markExchangeMessagesRead = async (
  id: string,
  lastMessageId: string,
): Promise<void> => {
  const payload = MarkReadRequestSchema.parse({
    last_message_id: lastMessageId,
  });

  await api.post(`/exchanges/${id}/messages/read`, payload);
};
