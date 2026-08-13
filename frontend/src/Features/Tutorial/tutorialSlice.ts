import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

type TutorialState = {
  active: boolean;
  step: number;
  userId: string | null;
};

const initialState: TutorialState = {
  active: false,
  step: 0,
  userId: null,
};

const tutorialSlice = createSlice({
  name: "tutorial",
  initialState,
  reducers: {
    tutorialStarted: (
      state,
      action: PayloadAction<{ step?: number; userId: string }>,
    ) => {
      state.active = true;
      state.step = action.payload.step ?? 0;
      state.userId = action.payload.userId;
    },
    tutorialAdvanced: (state) => {
      state.step += 1;
    },
    tutorialFinished: (state) => {
      state.active = false;
      state.step = 0;
      state.userId = null;
    },
    tutorialStopped: (state) => {
      state.active = false;
      state.step = 0;
      state.userId = null;
    },
  },
});

export const {
  tutorialAdvanced,
  tutorialFinished,
  tutorialStarted,
  tutorialStopped,
} = tutorialSlice.actions;

export default tutorialSlice.reducer;
