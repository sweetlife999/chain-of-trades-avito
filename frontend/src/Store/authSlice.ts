import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { TUser } from "../Api/auth/auth.types";


type AuthState = {
  user: TUser | null;
  isAuth: boolean;
};

const initialState: AuthState = {
  user: null,
  isAuth: false,
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    setUserState: (state, action: PayloadAction<TUser>) => {
      state.user = action.payload;
      state.isAuth = true;
    },

    logoutState: (state) => {
      state.user = null;
      state.isAuth = false;
    },
  },
});

export const { setUserState, logoutState } = authSlice.actions;

export default authSlice.reducer;