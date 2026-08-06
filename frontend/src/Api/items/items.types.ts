import z from "zod";

export const CategorySchema = z.object({
  name: z.string(),
  slug: z.string(),
});

export const CategoriesArraySchema = z.array(CategorySchema);

export const CreateItemRequestSchema = z.object({
  category: z.string().min(1),
  description: z.string(),
  photo_urls: z.array(z.string().url()).min(1),
  title: z.string().min(1),
  wants: z.array(z.string().min(1)).min(1),
});

export const ItemSchema = z.object({
  id: z.string(),
  owner_id: z.string(),
  title: z.string(),
  description: z.string(),
  category: z.string(),
  photo_urls: z.array(z.string()),
  wants: z.array(z.string()),
  status: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const ItemsArraySchema = z.array(ItemSchema);

export type TCategory = z.infer<typeof CategorySchema>;
export type TCategories = z.infer<typeof CategoriesArraySchema>;
export type TCreateItemRequest = z.infer<
  typeof CreateItemRequestSchema
>;
export type TItem = z.infer<typeof ItemSchema>;
export type TGetItems = z.infer<typeof ItemsArraySchema>;