import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ConfirmationPopup } from "./ConfirmationPopup";

const renderPopup = (overrides: Partial<React.ComponentProps<typeof ConfirmationPopup>> = {}) => {
  const onClose = vi.fn();
  const onConfirm = vi.fn();

  render(
    <ConfirmationPopup
      confirmLabel="Подтвердить"
      description="Действие нельзя отменить"
      title="Продолжить?"
      onClose={onClose}
      onConfirm={onConfirm}
      {...overrides}
    />,
  );

  return { onClose, onConfirm };
};

describe("ConfirmationPopup", () => {
  it("runs the confirmation callback", async () => {
    const user = userEvent.setup();
    const { onConfirm } = renderPopup();

    await user.click(screen.getByRole("button", { name: "Подтвердить" }));

    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it("lets the user refuse the action using the back button or Escape", async () => {
    const user = userEvent.setup();
    const { onClose } = renderPopup();

    await user.click(screen.getByRole("button", { name: "Назад" }));
    fireEvent.keyDown(window, { key: "Escape" });

    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it("blocks confirmation and closing while the action is pending", () => {
    const { onClose, onConfirm } = renderPopup({
      isPending: true,
      pendingLabel: "Сохраняем...",
    });

    expect(screen.getByRole("button", { name: "Назад" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Сохраняем..." })).toBeDisabled();

    fireEvent.keyDown(window, { key: "Escape" });

    expect(onClose).not.toHaveBeenCalled();
    expect(onConfirm).not.toHaveBeenCalled();
  });
});

