import axios from "axios";

import api from "../client";
import {
  AdminUserExchangesParamsSchema,
  AdminUserExchangesSchema,
  AdminReportDecisionSchema,
  AdminReportMessagesSchema,
  AdminReportSchema,
  AdminReportsParamsSchema,
  AdminReportsSchema,
  CreatePickupPointSchema,
  DashboardSchema,
  PickupPointSchema,
  PickupPointsSchema,
  UpdatePickupPointSchema,
  type TCreatePickupPoint,
  type TAdminUserExchanges,
  type TAdminUserExchangesParams,
  type TAdminReport,
  type TAdminReportDecision,
  type TAdminReportMessages,
  type TAdminReports,
  type TAdminReportsParams,
  type TDashboard,
  type TPickupPoint,
  type TUpdatePickupPoint,
} from "./admin.types";

export const getAdminErrorMessage = (
  error: unknown,
  fallback: string,
): string => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? fallback;
  }

  return fallback;
};

export const getAdminDashboard = async (): Promise<TDashboard> => {
  const { data } = await api.get("/admin/dashboard");

  return DashboardSchema.parse(data);
};

export const getAdminPickupPoints = async (): Promise<TPickupPoint[]> => {
  const { data } = await api.get("/admin/pickup-points");

  return PickupPointsSchema.parse(data);
};

export const getAdminPickupPoint = async (
  id: string,
): Promise<TPickupPoint> => {
  const { data } = await api.get(`/admin/pickup-points/${id}`);

  return PickupPointSchema.parse(data);
};

export const createAdminPickupPoint = async (
  request: TCreatePickupPoint,
): Promise<TPickupPoint> => {
  const payload = CreatePickupPointSchema.parse(request);
  const { data } = await api.post("/admin/pickup-points", payload);

  return PickupPointSchema.parse(data);
};

export const updateAdminPickupPoint = async (
  id: string,
  request: TUpdatePickupPoint,
): Promise<TPickupPoint> => {
  const payload = UpdatePickupPointSchema.parse(request);
  const { data } = await api.patch(`/admin/pickup-points/${id}`, payload);

  return PickupPointSchema.parse(data);
};

export const deleteAdminPickupPoint = async (id: string): Promise<void> => {
  await api.delete(`/admin/pickup-points/${id}`);
};

export const getAdminUserExchanges = async (
  userId: string,
  request: TAdminUserExchangesParams = {},
): Promise<TAdminUserExchanges> => {
  const params = AdminUserExchangesParamsSchema.parse(request);
  const { data } = await api.get(`/admin/users/${userId}/exchanges`, {
    params,
  });

  return AdminUserExchangesSchema.parse(data);
};

export const cancelAdminExchange = async (id: string): Promise<void> => {
  await api.post(`/admin/exchanges/${id}/cancel`);
};

export const markAdminExchangeDelivered = async (id: string): Promise<void> => {
  await api.post(`/admin/exchanges/${id}/mark-delivered`);
};

export const getAdminReports = async (
  request: TAdminReportsParams = {},
): Promise<TAdminReports> => {
  const params = AdminReportsParamsSchema.parse(request);
  const { data } = await api.get("/admin/reports", { params });

  return AdminReportsSchema.parse(data);
};

export const getAdminReport = async (id: string): Promise<TAdminReport> => {
  const { data } = await api.get(`/admin/reports/${id}`);

  return AdminReportSchema.parse(data);
};

export const assignAdminReport = async (id: string): Promise<TAdminReport> => {
  const { data } = await api.post(`/admin/reports/${id}/assign`);

  return AdminReportSchema.parse(data);
};

const decideAdminReport = async (
  id: string,
  decision: "reject" | "resolve",
  request: TAdminReportDecision,
): Promise<TAdminReport> => {
  const payload = AdminReportDecisionSchema.parse(request);
  const { data } = await api.post(
    `/admin/reports/${id}/${decision}`,
    payload,
  );

  return AdminReportSchema.parse(data);
};

export const rejectAdminReport = (
  id: string,
  request: TAdminReportDecision,
) => decideAdminReport(id, "reject", request);

export const resolveAdminReport = (
  id: string,
  request: TAdminReportDecision,
) => decideAdminReport(id, "resolve", request);

export const getAdminReportMessages = async (
  id: string,
): Promise<TAdminReportMessages> => {
  const { data } = await api.get(`/admin/reports/${id}/messages`);

  return AdminReportMessagesSchema.parse(data);
};
