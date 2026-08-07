import z from "zod";

export const formSchema = z.object({
  nickname: z.string().min(3, "Никнейм должен содержать не менее 3 символов").max(32, "Максимум 32 символа"),
  password: z.string().min(6, "Пароль должен содержать не менее 6 символов"),
});

export type FormState = z.infer<typeof formSchema>;