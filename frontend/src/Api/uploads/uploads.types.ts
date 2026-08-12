import { z } from "zod";

// Ссылку сервер отдаёт относительным путём: /uploads/<uuid>.<расширение>.
export const UploadResponseSchema = z.object({
  url: z.string().min(1),
});

export type TUploadResponse = z.infer<typeof UploadResponseSchema>;
