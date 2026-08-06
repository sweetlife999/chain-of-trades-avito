import api from "../client";

import { ExchangeSchema, ExchangesSchema, type TExchange, type TExchanges } from "./exchanges.types";




export const getExchanges = async (): Promise<TExchanges> => {
  const { data } = await api.get("/exchanges");
  return ExchangesSchema.parse(data);
};

export const getExchange = async (id: string): Promise<TExchange> => {
  const { data } = await api.get(`/exchanges/${id}`);
  return ExchangeSchema.parse(data);
};
