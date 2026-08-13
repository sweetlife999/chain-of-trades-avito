import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  closeSupportThread,
  createSupportThread,
  getSupportMessages,
  getSupportThreads,
  sendSupportMessage,
} from "../../../Api/support/support";
import { SupportPage } from "./SupportPage";
import { supportHints } from "./supportHints";

vi.mock("../../../Api/support/support", () => ({
  closeSupportThread: vi.fn(),
  createSupportThread: vi.fn(),
  getSupportError: (_error: unknown, fallback: string) => fallback,
  getSupportMessages: vi.fn(),
  getSupportThreads: vi.fn(),
  sendSupportMessage: vi.fn(),
}));

vi.mock("../../../Hooks/useAuthDispatch", () => ({
  useAuthSelector: () => ({
    isAuth: true,
    user: {
      experience: { level: 3 },
      id: "user-1",
      nickname: "Пользователь",
    },
  }),
}));

vi.mock("../../UI/Mascot/Mascot", () => ({
  Mascot: ({ level }: { level?: number }) => (
    <div data-level={level} data-testid="support-mascot" />
  ),
}));

const thread = {
  assigned_admin: null,
  closed_at: null,
  created_at: "2026-08-13T10:00:00.000Z",
  escalated_at: null,
  id: "thread-1",
  last_message: "",
  last_message_at: "2026-08-13T10:00:00.000Z",
  status: "open" as const,
  subject: "Проблема с обменом",
  unread_count: 0,
  updated_at: "2026-08-13T10:00:00.000Z",
  user: null,
};

const renderSupportPage = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/support?thread=thread-1"]}>
        <SupportPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("SupportPage assistant", () => {
  beforeEach(() => {
    vi.mocked(getSupportThreads).mockResolvedValue([thread]);
    vi.mocked(getSupportMessages).mockResolvedValue({
      messages: [],
      thread,
    });
    vi.mocked(sendSupportMessage).mockResolvedValue({
      author: {
        id: "user-1",
        is_admin: false,
        nickname: "Пользователь",
        photo_url: null,
      },
      body: supportHints[0],
      created_at: "2026-08-13T10:01:00.000Z",
      id: "message-1",
      thread_id: "thread-1",
    });
    vi.mocked(closeSupportThread).mockResolvedValue(thread);
    vi.mocked(createSupportThread).mockResolvedValue(thread);
  });

  it("shows Umi and inserts a selected hint into the message field", async () => {
    const user = userEvent.setup();
    renderSupportPage();

    const assistant = await screen.findByRole("region", {
      name: "Быстрые подсказки Уми",
    });
    const messageField = screen.getByRole("textbox", { name: "Сообщение" });

    expect(within(assistant).getByTestId("support-mascot")).toHaveAttribute(
      "data-level",
      "3",
    );
    expect(within(assistant).getAllByRole("button")).toHaveLength(
      supportHints.length,
    );

    await user.click(
      within(assistant).getByRole("button", { name: supportHints[0] }),
    );

    expect(messageField).toHaveValue(supportHints[0]);
    await waitFor(() => {
      expect(messageField).toHaveFocus();
    });
  });

  it("sends the selected hint with Enter", async () => {
    const user = userEvent.setup();
    renderSupportPage();

    const hintButton = await screen.findByRole("button", {
      name: supportHints[0],
    });
    await user.click(hintButton);
    await waitFor(() => {
      expect(screen.getByRole("textbox", { name: "Сообщение" })).toHaveFocus();
    });
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(vi.mocked(sendSupportMessage).mock.calls[0]?.[0]).toEqual({
        body: supportHints[0],
        threadID: "thread-1",
      });
    });
    expect(screen.getByRole("textbox", { name: "Сообщение" })).toHaveValue("");
  });
});
