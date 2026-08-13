import { describe, expect, it } from "vitest";

import reducer, {
  mascotReacted,
  mascotReset,
  mascotSettled,
} from "./mascotSlice";

describe("mascotSlice", () => {
  it("applies a reaction and increments its revision", () => {
    const state = reducer(undefined, mascotReacted("CHAIN_AVAILABLE"));

    expect(state.mood).toBe("excited");
    expect(state.mode).toBe("dialog");
    expect(state.message).toContain("Нашлась цепочка");
    expect(state.revision).toBe(1);
  });

  it("ignores a stale settle timer", () => {
    const reacted = reducer(undefined, mascotReacted("ERROR"));
    const state = reducer(
      reacted,
      mascotSettled(reacted.revision - 1),
    );

    expect(state).toEqual(reacted);
  });

  it("resets the active reaction", () => {
    const reacted = reducer(undefined, mascotReacted("EMPTY_CHAINS"));
    const state = reducer(reacted, mascotReset());

    expect(state).toMatchObject({
      durationMs: null,
      message: null,
      mode: "ambient",
      mood: "idle",
      revision: reacted.revision + 1,
    });
  });
});