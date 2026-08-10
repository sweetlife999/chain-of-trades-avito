import { z } from "zod";

import {
  ExchangeMessagesSchema,
  ExchangeParticipantSchema,
} from "../exchanges/exchanges.types";

const StatisticsNumberSchema = z.number().int().nonnegative();

export const DashboardSchema = z.object({
  exchanges: z.object({
    cancelled: StatisticsNumberSchema,
    completed: StatisticsNumberSchema,
    confirmed: StatisticsNumberSchema,
    delivered: StatisticsNumberSchema,
    delivering: StatisticsNumberSchema,
    proposed: StatisticsNumberSchema,
    total: StatisticsNumberSchema,
  }),
  items: z.object({
    available: StatisticsNumberSchema,
    reserved: StatisticsNumberSchema,
    total: StatisticsNumberSchema,
    traded: StatisticsNumberSchema,
    withdrawn: StatisticsNumberSchema,
  }),
  pickup_points_total: StatisticsNumberSchema,
  users_total: StatisticsNumberSchema,
});

export const PickupPointSchema = z.object({
  address: z.string(),
  created_at: z.string(),
  id: z.string(),
  name: z.string(),
  updated_at: z.string(),
});

export const PickupPointsSchema = z.array(PickupPointSchema);

export const AdminExchangeStatusSchema = z.enum([
  "proposed",
  "confirmed",
  "delivering",
  "delivered",
]);

export const AdminExchangeSchema = z.object({
  created_at: z.string(),
  id: z.string(),
  participants: z.array(ExchangeParticipantSchema),
  status: AdminExchangeStatusSchema,
  updated_at: z.string(),
});

export const AdminExchangesPaginationSchema = z.object({
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().nonnegative(),
  total: z.number().int().nonnegative(),
});

export const AdminExchangesSchema = z.object({
  exchanges: z.array(AdminExchangeSchema),
  pagination: AdminExchangesPaginationSchema,
});

export const AdminExchangesParamsSchema = z.object({
  status: AdminExchangeStatusSchema,
  limit: z.number().int().min(1).max(100).default(20),
  offset: z.number().int().nonnegative().default(0),
});

export const AdminUserExchangesSchema = AdminExchangesSchema;

export const AdminUserExchangesParamsSchema = z.object({
  limit: z.number().int().min(1).max(100).default(20),
  offset: z.number().int().nonnegative().default(0),
});

export const AdminReportStatusSchema = z.enum([
  "open",
  "resolved",
  "rejected",
]);

export const AdminReportReasonSchema = z.enum(["spam", "abuse", "other"]);

export const AdminReportUserSchema = z.object({
  id: z.string(),
  nickname: z.string(),
  photo_url: z.string().nullable().optional(),
});

export const AdminReportSchema = z.object({
  assigned_at: z.string().nullable().optional(),
  assignee: AdminReportUserSchema.nullable().optional(),
  closed_at: z.string().nullable().optional(),
  comment: z.string().nullable().optional(),
  created_at: z.string(),
  exchange: z.object({
    id: z.string(),
    status: z.string(),
  }),
  id: z.string(),
  message: z.object({
    body: z.string(),
    created_at: z.string(),
    id: z.string(),
  }),
  offender: AdminReportUserSchema,
  reason: AdminReportReasonSchema,
  reporter: AdminReportUserSchema,
  resolution_comment: z.string().nullable().optional(),
  status: AdminReportStatusSchema,
});

export const AdminReportsPaginationSchema = z.object({
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().nonnegative(),
  total: z.number().int().nonnegative(),
});

export const AdminReportsSchema = z.object({
  pagination: AdminReportsPaginationSchema,
  reports: z.array(AdminReportSchema),
});

export const AdminReportsParamsSchema = z.object({
  assignee_id: z.string().min(1).optional(),
  limit: z.number().int().min(1).max(100).default(20),
  offset: z.number().int().nonnegative().default(0),
  reason: AdminReportReasonSchema.optional(),
  status: AdminReportStatusSchema.optional(),
});

export const AdminReportDecisionSchema = z.object({
  comment: z.string().trim(),
});

export const AdminReportMessagesSchema = z.object({
  exchange_id: z.string(),
  messages: ExchangeMessagesSchema,
  report_id: z.string(),
});

export const AdminAuditEntrySchema = z.object({
  action: z.string(),
  admin_id: z.string(),
  created_at: z.string(),
  id: z.string(),
  metadata: z
    .record(z.string(), z.unknown())
    .nullable()
    .optional()
    .transform((value) => value ?? {}),
  target_id: z.string(),
  target_type: z.string(),
});

export const AdminAuditLogSchema = z.object({
  entries: z.array(AdminAuditEntrySchema),
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().nonnegative(),
  total: z.number().int().nonnegative(),
});

export const AdminAuditLogParamsSchema = z.object({
  admin_id: z.string().trim().min(1).optional(),
  action: z.string().trim().min(1).optional(),
  from: z.string().trim().min(1).optional(),
  to: z.string().trim().min(1).optional(),
  limit: z.number().int().min(1).max(100).default(20),
  offset: z.number().int().nonnegative().default(0),
});

export const AdminUserBlockResponseSchema = z.object({
  blocked_at: z.string().nullable().optional(),
  blocked_by: z.string().nullable().optional(),
  id: z.string(),
  is_blocked: z.boolean(),
  nickname: z.string(),
});

export const CreatePickupPointSchema = z.object({
  name: z.string().trim().min(1, "Укажите название ПВЗ"),
  address: z.string().trim().min(1, "Укажите адрес ПВЗ"),
});

export const UpdatePickupPointSchema = CreatePickupPointSchema.partial().refine(
  (request) => Object.keys(request).length > 0,
  { message: "Передайте хотя бы одно поле для изменения" },
);

export type TDashboard = z.infer<typeof DashboardSchema>;
export type TPickupPoint = z.infer<typeof PickupPointSchema>;
export type TCreatePickupPoint = z.infer<typeof CreatePickupPointSchema>;
export type TUpdatePickupPoint = z.infer<typeof UpdatePickupPointSchema>;
export type TAdminExchangeStatus = z.infer<typeof AdminExchangeStatusSchema>;
export type TAdminExchange = z.infer<typeof AdminExchangeSchema>;
export type TAdminExchanges = z.infer<typeof AdminExchangesSchema>;
export type TAdminExchangesParams = z.input<typeof AdminExchangesParamsSchema>;
export type TAdminUserExchanges = z.infer<typeof AdminUserExchangesSchema>;
export type TAdminUserExchangesParams = z.input<
  typeof AdminUserExchangesParamsSchema
>;
export type TAdminReportStatus = z.infer<typeof AdminReportStatusSchema>;
export type TAdminReportReason = z.infer<typeof AdminReportReasonSchema>;
export type TAdminReport = z.infer<typeof AdminReportSchema>;
export type TAdminReports = z.infer<typeof AdminReportsSchema>;
export type TAdminReportsParams = z.input<typeof AdminReportsParamsSchema>;
export type TAdminReportDecision = z.infer<typeof AdminReportDecisionSchema>;
export type TAdminReportMessages = z.infer<typeof AdminReportMessagesSchema>;
export type TAdminAuditEntry = z.infer<typeof AdminAuditEntrySchema>;
export type TAdminAuditLog = z.infer<typeof AdminAuditLogSchema>;
export type TAdminAuditLogParams = z.input<typeof AdminAuditLogParamsSchema>;
export type TAdminUserBlockResponse = z.infer<
  typeof AdminUserBlockResponseSchema
>;
