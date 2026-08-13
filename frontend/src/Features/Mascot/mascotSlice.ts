import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import { mascotReactions } from "./mascotEvents";
import type {
  MascotAnchor,
  MascotEvent,
  MascotMode,
  MascotMood,
  MascotMovement,
} from "./mascot.types";

type MascotState = {
  mood: MascotMood;
  mode: MascotMode;
  message: string | null;
  durationMs: number | null;
  anchor: MascotAnchor | null;
  movement: MascotMovement;
  revision: number;
};

const initialState: MascotState = {
  mood: "idle",
  mode: "ambient",
  message: null,
  durationMs: null,
  anchor: null,
  movement: "roam",
  revision: 0,
};

const mascotSlice = createSlice({
  name: "mascot",
  initialState,
  reducers: {
    mascotReacted: (state, action: PayloadAction<MascotEvent>) => {
      const reaction = mascotReactions[action.payload];
      state.mood = reaction.mood;
      state.mode = reaction.mode;
      state.message = reaction.message;
      state.durationMs = reaction.durationMs;
      state.anchor = reaction.anchor;
      state.movement = reaction.movement;
      state.revision += 1;
    },
    mascotSettled: (state, action: PayloadAction<number>) => {
      if (action.payload !== state.revision) {
        return;
      }

      state.mood = "idle";
      state.mode = "ambient";
      state.message = null;
      state.durationMs = null;
      state.anchor = null;
      state.movement = "roam";
    },
    mascotReset: (state) => {
      state.mood = "idle";
      state.mode = "ambient";
      state.message = null;
      state.durationMs = null;
      state.anchor = null;
      state.movement = "roam";
      state.revision += 1;
    },
  },
});

export const { mascotReacted, mascotReset, mascotSettled } = mascotSlice.actions;
export default mascotSlice.reducer;
