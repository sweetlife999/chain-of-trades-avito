import { z } from "zod";

// В ответах API отсутствие аватара кодируется пустой строкой (а в части старых
// ответов мог встречаться null). Это данные сервера, поэтому здесь не требуем URL:
// Swagger описывает photo_url просто как string.
export const PhotoUrlResponseSchema = z.string().nullable();
export const OptionalPhotoUrlResponseSchema = PhotoUrlResponseSchema.optional();

const HttpPhotoUrlSchema = z
  .url("Введите корректную ссылку")
  .regex(/^https?:\/\//, "Должна быть ссылка с http или https");

// В формах аватар необязателен: пустая строка означает «без фотографии».
// Если ссылка указана, принимаем только абсолютный http(s)-URL.
export const PhotoUrlInputSchema = z.union([
  z.literal(""),
  HttpPhotoUrlSchema,
]);
