import z from "zod";

// Фотографии приезжают в форму уже загруженными: PhotoUploader кладёт сюда ссылки,
// которые вернул сервер, поэтому проверять остаётся только их количество.
export const createChainFormSchema = z.object({
  title: z.string().trim().min(1, "Введите название товара"),
  category: z.string().min(1, "Выберите категорию"),
  description: z.string().trim().min(1, "Добавьте описание товара"),
  photo_urls: z
    .array(z.string())
    .min(1, "Добавьте хотя бы одну фотографию")
    .max(10, "Можно добавить не больше 10 фотографий"),
  wants: z.array(z.string()).min(1, "Выберите хотя бы одну желаемую категорию"),
  search_filters: z.object({
    max_chain_length: z.number().int().min(2).max(5),
    min_participant_rating: z.number().min(0).max(5),
    prefer_reliable_participants: z.boolean(),
  }),
});

export type TCreateChainForm = z.infer<typeof createChainFormSchema>;
