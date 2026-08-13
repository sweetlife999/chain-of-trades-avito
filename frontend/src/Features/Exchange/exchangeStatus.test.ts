import { describe, expect, it } from "vitest";

import { ExchangeStatusSchema } from "../../Api/exchanges/exchanges.types";
import { exchangeStatusPresentation } from "./exchangeStatus";

describe("exchangeStatusPresentation", () => {
  it("contains every status accepted by the API schema", () => {
    expect(Object.keys(exchangeStatusPresentation).sort()).toEqual(
      [...ExchangeStatusSchema.options].sort(),
    );
  });

  it("keeps all active exchange stages in the active tab", () => {
    expect(
      ["proposed", "confirmed", "delivering", "delivered"].map(
        (status) =>
          exchangeStatusPresentation[
            status as keyof typeof exchangeStatusPresentation
          ].tab,
      ),
    ).toEqual(["active", "active", "active", "active"]);
  });

  it("separates completed and cancelled exchanges", () => {
    expect(exchangeStatusPresentation.completed.tab).toBe("completed");
    expect(exchangeStatusPresentation.cancelled.tab).toBe("cancelled");
  });

  it("advances progress in the same order as the exchange lifecycle", () => {
    expect(
      ["proposed", "confirmed", "delivering", "delivered", "completed"].map(
        (status) =>
          exchangeStatusPresentation[
            status as keyof typeof exchangeStatusPresentation
          ].progressStep,
      ),
    ).toEqual([1, 2, 3, 4, 5]);
  });

  it("provides a non-empty label for every UI context", () => {
    Object.values(exchangeStatusPresentation).forEach((presentation) => {
      expect(presentation.adminLabel).not.toHaveLength(0);
      expect(presentation.detailsLabel).not.toHaveLength(0);
      expect(presentation.feedLabel).not.toHaveLength(0);
      expect(presentation.listLabel).not.toHaveLength(0);
    });
  });
});

