import { z } from "zod";

import { PhotoUrlInputSchema, PhotoUrlResponseSchema } from "../common.types";

const BaseUserSchema = z.object({
  created_at: z.string(),
  deals_broken: z.number().int().nonnegative(),
  deals_completed: z.number().int().nonnegative(),
  description: z.string(),
  id: z.string(),
  nickname: z.string(),
  photo_url: PhotoUrlResponseSchema,
  // По Swagger null означает, что оценок ещё нет. Это принципиально не 0.
  rating: z.number().nullable(),
  ratings_count: z.number().int().nonnegative(),
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
  photo_url: PhotoUrlResponseSchema,
});

export const BlockedUsersSchema = z.array(BlockedUserSchema);

export type TBlockedUser = z.infer<typeof BlockedUserSchema>;
export type TBlockedUsers = z.infer<typeof BlockedUsersSchema>;

export const registerSchema = z.object({
  nickname: z
    .string()
    .trim()
    .min(3, "Никнейм должен содержать минимум 3 символа")
    .max(32, "Никнейм должен содержать максимум 32 символа"),

  password: z
    .string()
    .min(8, "Пароль должен содержать минимум 8 символов")
    .max(72, "Пароль должен содержать максимум 72 символа"),

  description: z.string().trim(),

  photo_url: PhotoUrlInputSchema,
});

export const UpdateUserSchema = z.object({
  nickname: z.string().trim().min(3).max(32).optional(),
  description: z.string().trim().optional(),
  photo_url: PhotoUrlInputSchema.optional(),
});

export type TRegister = z.infer<typeof registerSchema>;
export type TUpdateUser = z.infer<typeof UpdateUserSchema>;
