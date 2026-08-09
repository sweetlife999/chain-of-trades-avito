import axios from "axios";

import api from "../client";
import {
  CreateReportRequestSchema,
  ReportSchema,
  type TCreateReportRequest,
  type TReport,
} from "./reports.types";

export const createReport = async (
  request: TCreateReportRequest,
): Promise<TReport> => {
  const payload = CreateReportRequestSchema.parse(request);
  const { data } = await api.post("/reports", payload);

  return ReportSchema.parse(data);
};

export const getReportErrorMessage = (error: unknown): string => {
  if (!axios.isAxiosError<{ error?: string }>(error)) {
    return "Не удалось отправить жалобу. Повторите попытку.";
  }

  switch (error.response?.status) {
    case 400:
      return error.response.data?.error ?? "Проверьте причину и комментарий.";
    case 403:
      return "На это сообщение нельзя пожаловаться.";
    case 404:
      return "Сообщение больше недоступно.";
    case 409:
      return "Вы уже жаловались на это сообщение.";
    default:
      return (
        error.response?.data?.error ??
        "Не удалось отправить жалобу. Повторите попытку."
      );
  }
};
