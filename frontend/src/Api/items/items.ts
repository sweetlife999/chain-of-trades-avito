import axios from "axios";

import api from "../client";
import {
  CategoriesArraySchema,
  CreateItemRequestSchema,
  ItemsArraySchema,
  ItemSchema,
  SetPickupPointRequestSchema,
  UpdateItemRequestSchema,
  type TCategories,
  type TCreateItemRequest,
  type TGetItems,
  type TItem,
  type TSetPickupPointRequest,
  type TUpdateItemRequest,
} from "./items.types";

export const getItems = async (): Promise<TGetItems> => {
  const { data } = await api.get("/items");
  return ItemsArraySchema.parse(data);
};

export const getItem = async (id: string): Promise<TItem> => {
  const { data } = await api.get(`/items/${id}`);
  return ItemSchema.parse(data);
};

export const getCategories = async (): Promise<TCategories> => {
  const { data } = await api.get("/categories");
  return CategoriesArraySchema.parse(data);
};

export const createItem = async (request: TCreateItemRequest): Promise<TItem> => {
  const payload = CreateItemRequestSchema.parse(request);
  const { data } = await api.post("/items", payload);
  return ItemSchema.parse(data);
};

export const updateItem = async (
  id: string,
  request: TUpdateItemRequest,
): Promise<TItem> => {
  const payload = UpdateItemRequestSchema.parse(request);
  const { data } = await api.patch(`/items/${id}`, payload);
  return ItemSchema.parse(data);
};

export const deleteItem = async (id: string): Promise<void> => {
  await api.delete(`/items/${id}`);
};

export const restoreItemToSearch = async (id: string): Promise<TItem> => {
  const { data } = await api.put(`/items/${id}/search`);
  return ItemSchema.parse(data);
};

export const withdrawItemFromSearch = async (id: string): Promise<TItem> => {
  const { data } = await api.delete(`/items/${id}/search`);
  return ItemSchema.parse(data);
};

export const setItemPickupPoint = async (
  id: string,
  request: TSetPickupPointRequest,
): Promise<void> => {
  const payload = SetPickupPointRequestSchema.parse(request);

  await api.post(`/items/${id}/pickup`, payload);
};

export const removeItemFromPickupPoint = async (id: string): Promise<void> => {
  await api.delete(`/items/${id}/pickup`);
};

export const getItemErrorMessage = (
  error: unknown,
  fallback: string,
): string => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? fallback;
  }

  return fallback;
};
