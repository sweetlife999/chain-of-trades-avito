import z from "zod";

export const photoUrlSchema = z
  .url("Введите корректную ссылку")
  .regex(/^https?:\/\//, "Должна быть ссылка с http или https")
  .min(1, "Добавьте ссылку на фотографию");

const photoSchema = z.object({
  url: photoUrlSchema,
});

export const createChainFormSchema = z.object({
  title: z.string().trim().min(1, "Введите название товара"),
  category: z.string().min(1, "Выберите категорию"),
  description: z.string().trim().min(1, "Добавьте описание товара"),
  photo_urls: z
    .array(photoSchema)
    .min(1, "Добавьте хотя бы одну фотографию")
    .max(10, "Можно добавить не больше 10 фотографий"),
  wants: z.array(z.string()).min(1, "Выберите хотя бы одну желаемую категорию"),
});

export type TCreateChainForm = z.infer<typeof createChainFormSchema>;
