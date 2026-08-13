import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  completeExchange,
  confirmExchange,
  declineExchange,
  getExchange,
} from "../../../../Api/exchanges/exchanges";
import { ExchangeSchema } from "../../../../Api/exchanges/exchanges.types";
import { ExchangeDetails } from "./ExchangeDetails";

vi.mock("../../../../Api/exchanges/exchanges", () => ({
  completeExchange: vi.fn(),
  confirmExchange: vi.fn(),
  declineExchange: vi.fn(),
  getExchange: vi.fn(),
}));

vi.mock("../../../../Hooks/useAuthDispatch", () => ({
  useAuthSelector: () => ({
    isAdmin: false,
    isAuth: true,
    user: {
      id: "user-1",
      is_admin: false,
      nickname: "Пользователь",
    },
  }),
}));

vi.mock("../ExchangeChat/ExchangeChat", () => ({
  ExchangeChat: () => null,
}));

const proposedExchange = ExchangeSchema.parse({
  created_at: "2026-08-13T10:00:00.000Z",
  id: "exchange-1",
  participants: [
    {
      completion_confirmed_at: null,
      decided_at: null,
      gives_item: {
        category: { name: "Техника", slug: "electronics" },
        description: "Описание телефона",
        id: "item-1",
        pickup_point: null,
        status: "reserved",
        title: "Телефон",
      },
      position: 1,
      receives_item: {
        category: { name: "Книги", slug: "books" },
        description: "Описание книги",
        id: "item-2",
        pickup_point: null,
        status: "reserved",
        title: "Книга",
      },
      status: "pending",
      user: {
        id: "user-1",
        nickname: "Пользователь",
        photo_url: null,
      },
    },
  ],
  status: "proposed",
  updated_at: "2026-08-13T10:00:00.000Z",
});

const renderExchangeDetails = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/exchanges/exchange-1"]}>
        <Routes>
          <Route path="/exchanges/:id" element={<ExchangeDetails />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("ExchangeDetails decisions", () => {
  beforeEach(() => {
    vi.mocked(getExchange).mockResolvedValue(proposedExchange);
    vi.mocked(confirmExchange).mockResolvedValue(undefined);
    vi.mocked(declineExchange).mockResolvedValue(undefined);
    vi.mocked(completeExchange).mockResolvedValue(undefined);
  });

  it("confirms participation through the primary action", async () => {
    const user = userEvent.setup();
    renderExchangeDetails();

    await user.click(
      await screen.findByRole("button", { name: "Подтвердить" }),
    );

    await waitFor(() => {
      expect(confirmExchange).toHaveBeenCalledWith("exchange-1");
    });
    expect(declineExchange).not.toHaveBeenCalled();
  });

  it("declines participation only after explicit confirmation", async () => {
    const user = userEvent.setup();
    renderExchangeDetails();

    await user.click(
      await screen.findByRole("button", { name: "Отказаться" }),
    );

    expect(declineExchange).not.toHaveBeenCalled();
    expect(
      screen.getByRole("dialog", { name: "Отказаться от обмена?" }),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Да, отказаться" }),
    );

    await waitFor(() => {
      expect(declineExchange).toHaveBeenCalledWith("exchange-1");
    });
    expect(confirmExchange).not.toHaveBeenCalled();
  });
});

