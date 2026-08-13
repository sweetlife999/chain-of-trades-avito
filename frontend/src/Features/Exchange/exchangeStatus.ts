import type { TExchangeStatus } from "../../Api/exchanges/exchanges.types";

export type TExchangeStatusTab = "active" | "completed" | "cancelled";

type TExchangeStatusPresentation = {
  adminLabel: string;
  detailsLabel: string;
  feedLabel: string;
  listLabel: string;
  progressStep: number;
  tab: TExchangeStatusTab;
};

export const exchangeStatusPresentation = {
  proposed: {
    adminLabel: "Ожидает подтверждения",
    detailsLabel: "Ждём подтверждения",
    feedLabel: "Предложение обмена",
    listLabel: "Ждём вашего подтверждения",
    progressStep: 1,
    tab: "active",
  },
  confirmed: {
    adminLabel: "Подтверждён",
    detailsLabel: "Передача вещи в ПВЗ",
    feedLabel: "Обмен подтверждён",
    listLabel: "Передача вещи в ПВЗ",
    progressStep: 2,
    tab: "active",
  },
  delivering: {
    adminLabel: "Доставляется",
    detailsLabel: "Доставка между ПВЗ",
    feedLabel: "Вещи доставляются",
    listLabel: "Вещи доставляются между ПВЗ",
    progressStep: 3,
    tab: "active",
  },
  delivered: {
    adminLabel: "Ожидает получения",
    detailsLabel: "Вещь ожидает получения",
    feedLabel: "Вещь ожидает в ПВЗ",
    listLabel: "Вещь ожидает вас в ПВЗ",
    progressStep: 4,
    tab: "active",
  },
  completed: {
    adminLabel: "Завершён",
    detailsLabel: "Обмен завершён",
    feedLabel: "Обмен завершён",
    listLabel: "Обмен завершён",
    progressStep: 5,
    tab: "completed",
  },
  cancelled: {
    adminLabel: "Отменён",
    detailsLabel: "Цепочка распалась",
    feedLabel: "Обмен отменён",
    listLabel: "Цепочка распалась",
    progressStep: 1,
    tab: "cancelled",
  },
} as const satisfies Record<TExchangeStatus, TExchangeStatusPresentation>;

