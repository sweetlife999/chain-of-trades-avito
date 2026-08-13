import axios from "axios";

import api from "../client";
import {
  AdminAntiscamCaseSchema,
  AdminAntiscamCasesSchema,
  AdminAntiscamMessagesSchema,
  AdminAntiscamParamsSchema,
  AdminAuditLogParamsSchema,
  AdminAuditLogSchema,
  AdminExchangesParamsSchema,
  AdminExchangesSchema,
  AdminReportDecisionSchema,
  AdminReportMessagesSchema,
  AdminReportSchema,
  AdminReportsParamsSchema,
  AdminReportsSchema,
  AdminUserBlockResponseSchema,
  AdminUserExchangesParamsSchema,
  AdminUserExchangesSchema,
  CreatePickupPointSchema,
  DashboardSchema,
  PickupPointSchema,
  PickupPointsSchema,
  UpdatePickupPointSchema,
  type TAdminAuditLog,
  type TAdminAuditLogParams,
  type TAdminAntiscamCase,
  type TAdminAntiscamCases,
  type TAdminAntiscamMessages,
  type TAdminAntiscamParams,
  type TAdminExchanges,
  type TAdminExchangesParams,
  type TAdminReport,
  type TAdminReportDecision,
  type TAdminReportMessages,
  type TAdminReports,
  type TAdminReportsParams,
  type TAdminUserBlockResponse,
  type TAdminUserExchanges,
  type TAdminUserExchangesParams,
  type TCreatePickupPoint,
  type TDashboard,
  type TPickupPoint,
  type TUpdatePickupPoint,
} from "./admin.types";

export const getAdminAntiscamCases = async (
  request: TAdminAntiscamParams = {},
): Promise<TAdminAntiscamCases> => {
  const params = AdminAntiscamParamsSchema.parse(request);
  const { data } = await api.get("/admin/antiscam/cases", { params });
  return AdminAntiscamCasesSchema.parse(data);
};

export const getAdminAntiscamCase = async (
  id: string,
): Promise<TAdminAntiscamCase> => {
  const { data } = await api.get(`/admin/antiscam/cases/${id}`);
  return AdminAntiscamCaseSchema.parse(data);
};

export const getAdminAntiscamMessages = async (
  id: string,
): Promise<TAdminAntiscamMessages> => {
  const { data } = await api.get(`/admin/antiscam/cases/${id}/messages`);
  return AdminAntiscamMessagesSchema.parse(data);
};

const decideAdminAntiscamCase = async (
  id: string,
  action: "confirm" | "dismiss",
  comment: string,
): Promise<TAdminAntiscamCase> => {
  const { data } = await api.post(`/admin/antiscam/cases/${id}/${action}`, {
    comment,
  });
  return AdminAntiscamCaseSchema.parse(data);
};

export const confirmAdminAntiscamCase = (id: string, comment: string) =>
  decideAdminAntiscamCase(id, "confirm", comment);
export const dismissAdminAntiscamCase = (id: string, comment: string) =>
  decideAdminAntiscamCase(id, "dismiss", comment);

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

export const getAdminExchanges = async (
  request: TAdminExchangesParams,
): Promise<TAdminExchanges> => {
  const params = AdminExchangesParamsSchema.parse(request);
  const { data } = await api.get("/admin/exchanges", { params });

  return AdminExchangesSchema.parse(data);
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

export const getAdminAuditLog = async (
  request: TAdminAuditLogParams = {},
): Promise<TAdminAuditLog> => {
  const params = AdminAuditLogParamsSchema.parse(request);
  const { data } = await api.get("/admin/audit-log", { params });

  return AdminAuditLogSchema.parse(data);
};

export const blockAdminUser = async (
  id: string,
): Promise<TAdminUserBlockResponse> => {
  const { data } = await api.post(`/admin/users/${id}/block`);

  return AdminUserBlockResponseSchema.parse(data);
};

export const unblockAdminUser = async (
  id: string,
): Promise<TAdminUserBlockResponse> => {
  const { data } = await api.post(`/admin/users/${id}/unblock`);

  return AdminUserBlockResponseSchema.parse(data);
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
  const { data } = await api.post(`/admin/reports/${id}/${decision}`, payload);

  return AdminReportSchema.parse(data);
};

export const rejectAdminReport = (id: string, request: TAdminReportDecision) =>
  decideAdminReport(id, "reject", request);

export const resolveAdminReport = (id: string, request: TAdminReportDecision) =>
  decideAdminReport(id, "resolve", request);

export const getAdminReportMessages = async (
  id: string,
): Promise<TAdminReportMessages> => {
  const { data } = await api.get(`/admin/reports/${id}/messages`);

  return AdminReportMessagesSchema.parse(data);
};
