import api from "../client";
import {
  ItemsArraySchema,
  type TGetItems,
} from "./items.types";

export const getItems = async (): Promise<TGetItems> => {
  const {data} = await api.get("/items");

  return ItemsArraySchema.parse(data);
};