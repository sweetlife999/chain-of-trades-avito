import { z } from "zod";

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

export type TItem = z.infer<typeof ItemSchema>;
export type TGetItems = z.infer<typeof ItemsArraySchema>;