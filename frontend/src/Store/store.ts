import { configureStore } from "@reduxjs/toolkit";
import authReducer from './authSlice'
import mascotReducer from '../Features/Mascot/mascotSlice'


export const store = configureStore({
    reducer: {
        auth: authReducer,
        mascot: mascotReducer
    }
})

export type RootState = ReturnType<typeof store.getState>
export type AuthDispatch = typeof store.dispatch