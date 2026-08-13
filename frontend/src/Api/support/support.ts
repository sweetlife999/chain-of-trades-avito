import axios from "axios";

import api from "../client";
import {
  AdminSupportPageSchema,
  SupportMessageSchema,
  SupportThreadMessagesSchema,
  SupportThreadSchema,
  type TAdminSupportPage,
  type TSupportMessage,
  type TSupportThread,
  type TSupportThreadMessages,
} from "./support.types";

export const getSupportError = (error: unknown, fallback: string) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? fallback;
  }
  return fallback;
};

export const getSupportThreads = async (): Promise<TSupportThread[]> => {
  const { data } = await api.get("/support/threads");
  return SupportThreadSchema.array().parse(data);
};

export const createSupportThread = async (request: {
  subject: string;
  body: string;
}): Promise<TSupportThread> => {
  const { data } = await api.post("/support/threads", request);
  return SupportThreadSchema.parse(data);
};

export const getSupportMessages = async (
  threadID: string,
): Promise<TSupportThreadMessages> => {
  const { data } = await api.get(`/support/threads/${threadID}/messages`);
  return SupportThreadMessagesSchema.parse(data);
};

export const sendSupportMessage = async ({
  threadID,
  body,
}: {
  threadID: string;
  body: string;
}): Promise<TSupportMessage> => {
  const { data } = await api.post(`/support/threads/${threadID}/messages`, { body });
  return SupportMessageSchema.parse(data);
};

export const closeSupportThread = async (
  threadID: string,
): Promise<TSupportThread> => {
  const { data } = await api.post(`/support/threads/${threadID}/close`);
  return SupportThreadSchema.parse(data);
};

export const getAdminSupportThreads = async (params: {
  status?: string;
  /** Только ждущие модератора: эскалированные и те, на которые никто не ответил. */
  needs_human?: boolean;
  limit?: number;
  offset?: number;
} = {}): Promise<TAdminSupportPage> => {
  const { data } = await api.get("/admin/support/threads", { params });
  return AdminSupportPageSchema.parse(data);
};

export const getAdminSupportMessages = async (
  threadID: string,
): Promise<TSupportThreadMessages> => {
  const { data } = await api.get(`/admin/support/threads/${threadID}/messages`);
  return SupportThreadMessagesSchema.parse(data);
};

export const assignAdminSupportThread = async (
  threadID: string,
): Promise<TSupportThread> => {
  const { data } = await api.post(`/admin/support/threads/${threadID}/assign`);
  return SupportThreadSchema.parse(data);
};

export const sendAdminSupportMessage = async ({
  threadID,
  body,
}: {
  threadID: string;
  body: string;
}): Promise<TSupportMessage> => {
  const { data } = await api.post(`/admin/support/threads/${threadID}/messages`, { body });
  return SupportMessageSchema.parse(data);
};

export const closeAdminSupportThread = async (
  threadID: string,
): Promise<TSupportThread> => {
  const { data } = await api.post(`/admin/support/threads/${threadID}/close`);
  return SupportThreadSchema.parse(data);
};
