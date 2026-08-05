import api from "../client";
import { UserSchema, type TRegister, type TUser } from "./auth.types";

type TLogin = {
  nickname: string;
  password: string;
};

export const login = async (loginData: TLogin): Promise<TUser> => {
  const { data } = await api.post("/auth/login", loginData);

  return UserSchema.parse(data);
};

export const logout = async (): Promise<void> => {
  await api.post("/auth/logout");
};

export const registerUser = async (
  data: TRegister,
): Promise<TUser> => {
  const response = await api.post("/users", data);

  return UserSchema.parse(response.data);
};

export const getMe = async (): Promise<TUser> => {
  const { data } = await api.get("/auth/me");

  return UserSchema.parse(data);
};



