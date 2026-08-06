import api from "../client";
import {
  CategoriesArraySchema,
  CreateItemRequestSchema,
  ItemsArraySchema,
  ItemSchema,
  type TCategories,
  type TCreateItemRequest,
  type TGetItems,
  type TItem,
} from "./items.types";

export const getItems = async (): Promise<TGetItems> => {
  const {data} = await api.get("/items");

  return ItemsArraySchema.parse(data);
};

export const getCategories = async (): Promise<TCategories> => {
  const { data } = await api.get("/categories");

  return CategoriesArraySchema.parse(data);
};

export const createItem = async (
  request: TCreateItemRequest,
): Promise<TItem> => {
  const payload = CreateItemRequestSchema.parse(request);
  const { data } = await api.post("/items", payload);

  return ItemSchema.parse(data);
};