import api from "../client";
import {
  BlockedUsersSchema,
  UpdateUserSchema,
  UserSchema,
  type TBlockedUsers,
  type TRegister,
  type TUpdateUser,
  type TUser,
} from "./auth.types";

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

export const registerUser = async (data: TRegister): Promise<TUser> => {
  const response = await api.post("/users", data);

  return UserSchema.parse(response.data);
};

export const registerAndLogin = async (data: TRegister): Promise<TUser> => {
  await registerUser(data);

  return login({
    nickname: data.nickname,
    password: data.password,
  });
};

export const getMe = async (): Promise<TUser> => {
  const { data } = await api.get("/auth/me");

  return UserSchema.parse(data);
};

export const getUserById = async (id: string): Promise<TUser> => {
  const { data } = await api.get(`/users/${id}`);

  return UserSchema.parse(data);
};

export const updateUser = async (
  id: string,
  request: TUpdateUser,
): Promise<TUser> => {
  const payload = UpdateUserSchema.parse(request);
  const { data } = await api.patch(`/users/${id}`, payload);

  return UserSchema.parse(data);
};


export const getBlockedUsers = async (): Promise<TBlockedUsers> => {
  const { data } = await api.get("/users/me/blocks");

  return BlockedUsersSchema.parse(data);
};

export const blockUser = async (userId: string): Promise<void> => {
  await api.post(`/users/me/blocks/${userId}`);
};

export const unblockUser = async (userId: string): Promise<void> => {
  await api.delete(`/users/me/blocks/${userId}`);
};
