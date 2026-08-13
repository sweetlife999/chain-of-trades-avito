import { configureStore } from "@reduxjs/toolkit";
import authReducer from "./authSlice";
import mascotReducer from "../Features/Mascot/mascotSlice";
import tutorialReducer from "../Features/Tutorial/tutorialSlice";

export const store = configureStore({
  reducer: {
    auth: authReducer,
    mascot: mascotReducer,
    tutorial: tutorialReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AuthDispatch = typeof store.dispatch;
