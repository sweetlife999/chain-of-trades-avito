import api from "../client";
import {
  PickupPointSchema,
  PickupPointsSchema,
  type TPickupPoint,
  type TPickupPoints,
} from "./pickupPoints.types";

export const getPickupPoints = async (): Promise<TPickupPoints> => {
  const { data } = await api.get("/pickup-points");

  return PickupPointsSchema.parse(data);
};

export const getPickupPoint = async (id: string): Promise<TPickupPoint> => {
  const { data } = await api.get(`/pickup-points/${id}`);

  return PickupPointSchema.parse(data);
};
