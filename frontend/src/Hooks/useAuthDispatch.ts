import { useDispatch, useSelector } from "react-redux";
import type { AuthDispatch, RootState } from "../Store/store";

export const useAuthDispatch = () => useDispatch<AuthDispatch>()

export const useAuthSelector = () => useSelector((state: RootState) => state.auth)