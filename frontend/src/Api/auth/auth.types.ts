import { z } from "zod";

export const UserSchema = z.object({
  created_at: z.string(),
  deals_broken: z.number(),
  deals_completed: z.number(),
  description: z.string(),
  id: z.string(),
  nickname: z.string(),
  photo_url: z.string(),
  rating: z.number().nullable().transform((value) => value ?? 0),
  updated_at: z.string(),
});


export type TUser = z.infer<typeof UserSchema>;

const getByteLength = (value: string) => {
  return new TextEncoder().encode(value).length;
};

export const registerSchema = z.object({
  nickname: z
    .string()
    .trim()
    .min(3, "Никнейм должен содержать минимум 3 символа")
    .max(32, "Никнейм должен содержать максимум 32 символа"),

  password: z
    .string()
    .refine(
      (value) => getByteLength(value) >= 8,
      "Пароль должен содержать минимум 8 байт",
    )
    .refine(
      (value) => getByteLength(value) <= 72,
      "Пароль должен содержать максимум 72 байта",
    ),

  description: z.string().trim(),

  photo_url: z
    .string()
    .trim()
    .url("Введите корректную ссылку")
    .or(z.literal("")),
});

export type TRegister = z.infer<typeof registerSchema>;