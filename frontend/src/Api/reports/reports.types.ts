import { z } from "zod";

export const CreateReportRequestSchema = z.object({
  comment: z.string().trim().optional(),
  message_id: z.string().min(1),
  reason: z.string().trim().min(1, "Укажите причину жалобы"),
});

export const ReportSchema = z.object({
  comment: z.string().nullable().optional(),
  created_at: z.string(),
  id: z.string(),
  message_id: z.string(),
  reason: z.string(),
  status: z.string(),
});

export type TCreateReportRequest = z.input<
  typeof CreateReportRequestSchema
>;
export type TReport = z.infer<typeof ReportSchema>;
