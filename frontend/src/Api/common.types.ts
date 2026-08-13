import { z } from "zod";

// В ответах API отсутствие аватара кодируется пустой строкой (а в части старых
// ответов мог встречаться null). Это данные сервера, поэтому здесь не требуем URL:
// Swagger описывает photo_url просто как string.
export const PhotoUrlResponseSchema = z.string().nullable();
export const OptionalPhotoUrlResponseSchema = PhotoUrlResponseSchema.optional();

const HttpPhotoUrlSchema = z
  .url("Введите корректную ссылку")
  .regex(/^https?:\/\//, "Должна быть ссылка с http или https");

// Путь, который вернул POST /uploads. Он относительный, поэтому z.url() его не примет.
const UploadedPhotoUrlSchema = z
  .string()
  .startsWith("/uploads/", "Загрузите фотографию заново");

// Ссылка на фотографию в запросах: файл, загруженный через POST /uploads, либо внешний
// http(s)-адрес — с такими живут объявления и профили, заведённые до загрузки файлов.
export const PhotoReferenceSchema = z.union([
  UploadedPhotoUrlSchema,
  HttpPhotoUrlSchema,
]);

// В формах аватар необязателен: пустая строка означает «без фотографии».
export const PhotoUrlInputSchema = z.union([
  z.literal(""),
  PhotoReferenceSchema,
]);
