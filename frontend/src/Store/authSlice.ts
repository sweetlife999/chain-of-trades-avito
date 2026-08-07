import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { TAuthenticatedUser } from "../Api/auth/auth.types";


type AuthState = {
  user: TAuthenticatedUser | null;
  isAuth: boolean;
  isAdmin: boolean;
};

const initialState: AuthState = {
  user: null,
  isAuth: false,
  isAdmin: false,
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    setUserState: (state, action: PayloadAction<TAuthenticatedUser>) => {
      state.user = action.payload;
      state.isAuth = true;
      state.isAdmin = action.payload.is_admin;
    },

    logoutState: (state) => {
      state.user = null;
      state.isAuth = false;
      state.isAdmin = false;
    },
  },
});

export const { setUserState, logoutState } = authSlice.actions;

export default authSlice.reducer;