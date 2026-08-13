import { z } from "zod";

import { PhotoReferenceSchema } from "../common.types";

export const ItemStatusSchema = z.enum([
  "available",
  "reserved",
  "traded",
  "withdrawn",
]);

export const CategorySchema = z.object({
  name: z.string(),
  slug: z.string(),
});

export const CategoriesArraySchema = z.array(CategorySchema);

export const SearchFiltersSchema = z.object({
  max_chain_length: z.number().int().min(2).max(5),
  min_participant_rating: z.number().min(0).max(5),
  prefer_reliable_participants: z.boolean(),
});

export const CreateItemRequestSchema = z.object({
  category: z.string().min(1),
  description: z.string(),
  photo_urls: z.array(PhotoReferenceSchema).min(1),
  title: z.string().min(1),
  wants: z.array(z.string().min(1)).min(1),
  search_filters: SearchFiltersSchema,
});

export const UpdateItemRequestSchema = CreateItemRequestSchema.partial().refine(
  (request) => Object.keys(request).length > 0,
  { message: "Передайте хотя бы одно поле для изменения" },
);

export const ItemPickupPointSchema = z.object({
  address: z.string(),
  id: z.string(),
  name: z.string(),
});

export const SetPickupPointRequestSchema = z.object({
  pickup_point_id: z.string().min(1),
});

export const ItemSchema = z.object({
  id: z.string(),
  owner_id: z.string(),
  title: z.string(),
  description: z.string(),
  category: z.string(),
  // Это данные сервера, а не форма: он их уже проверил, и ссылку, которую сам же и выдал,
  // мы обязаны показать. Строгая проверка формы уронила бы всю карточку вместо картинки —
  // ровно так же, как раньше падал профиль без аватарки.
  photo_urls: z.array(z.string()),
  pickup_point: ItemPickupPointSchema.nullable().optional().default(null),
  wants: z.array(z.string()),
  search_filters: SearchFiltersSchema,
  status: ItemStatusSchema,
  created_at: z.string(),
  updated_at: z.string(),
});

export const ItemsArraySchema = z.array(ItemSchema);

export type TItemStatus = z.infer<typeof ItemStatusSchema>;
export type TCategory = z.infer<typeof CategorySchema>;
export type TSearchFilters = z.infer<typeof SearchFiltersSchema>;
export type TCategories = z.infer<typeof CategoriesArraySchema>;
export type TCreateItemRequest = z.infer<typeof CreateItemRequestSchema>;
export type TUpdateItemRequest = z.infer<typeof UpdateItemRequestSchema>;
export type TSetPickupPointRequest = z.infer<
  typeof SetPickupPointRequestSchema
>;
export type TItemPickupPoint = z.infer<typeof ItemPickupPointSchema>;
export type TItem = z.infer<typeof ItemSchema>;
export type TGetItems = z.infer<typeof ItemsArraySchema>;
