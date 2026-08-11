import { z } from "zod";

const BaseUserSchema = z.object({
  created_at: z.string(),
  deals_broken: z.number(),
  deals_completed: z.number(),
  description: z.string(),
  id: z.string(),
  nickname: z.string(),
  photo_url: z.string(),
  // null означает «оценок ещё не было», и схлопывать его в 0 нельзя: на экране это
  // превращало новичка в человека с нулевым рейтингом.
  rating: z.number().nullable(),
  ratings_count: z.number().int().nonnegative().default(0),
  updated_at: z.string(),
});

export const UserSchema = BaseUserSchema;

export const AuthenticatedUserSchema = BaseUserSchema.extend({
  is_admin: z.boolean(),
});

export type TUser = z.infer<typeof UserSchema>;
export type TAuthenticatedUser = z.infer<typeof AuthenticatedUserSchema>;

export const BlockedUserSchema = z.object({
  blocked_at: z.string(),
  id: z.string(),
  nickname: z.string(),
  photo_url: z.string(),
});

export const BlockedUsersSchema = z.array(BlockedUserSchema);

export type TBlockedUser = z.infer<typeof BlockedUserSchema>;
export type TBlockedUsers = z.infer<typeof BlockedUsersSchema>;

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

export const UpdateUserSchema = z.object({
  nickname: z.string().trim().min(3).max(32).optional(),
  description: z.string().trim().optional(),
  photo_url: z.string().trim().optional(),
});

export type TRegister = z.infer<typeof registerSchema>;
export type TUpdateUser = z.infer<typeof UpdateUserSchema>;
