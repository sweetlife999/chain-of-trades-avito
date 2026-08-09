import { z } from "zod";

import { ExchangeParticipantSchema } from "../exchanges/exchanges.types";

const StatisticsNumberSchema = z.number().int().nonnegative();

export const DashboardSchema = z.object({
  exchanges: z.object({
    cancelled: StatisticsNumberSchema,
    completed: StatisticsNumberSchema,
    confirmed: StatisticsNumberSchema,
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

export const AdminExchangeSchema = z.object({
  created_at: z.string(),
  id: z.string(),
  participants: z.array(ExchangeParticipantSchema),
  status: z.enum(["proposed", "confirmed"]),
  updated_at: z.string(),
});

export const AdminExchangesPaginationSchema = z.object({
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().nonnegative(),
  total: z.number().int().nonnegative(),
});

export const AdminUserExchangesSchema = z.object({
  exchanges: z.array(AdminExchangeSchema),
  pagination: AdminExchangesPaginationSchema,
});

export const AdminUserExchangesParamsSchema = z.object({
  limit: z.number().int().min(1).max(100).default(20),
  offset: z.number().int().nonnegative().default(0),
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
export type TAdminExchange = z.infer<typeof AdminExchangeSchema>;
export type TAdminUserExchanges = z.infer<typeof AdminUserExchangesSchema>;
export type TAdminUserExchangesParams = z.input<
  typeof AdminUserExchangesParamsSchema
>;
