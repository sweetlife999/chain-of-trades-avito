import api from "../client";

import { ExchangesSchema, type TExchanges } from "./exchanges.types";

export const getExchanges = async (): Promise<TExchanges> => {
  const { data } = await api.get("/exchanges");

  return ExchangesSchema.parse(data);
};
