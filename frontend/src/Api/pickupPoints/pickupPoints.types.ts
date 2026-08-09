import { z } from "zod";

export const PickupPointSchema = z.object({
  address: z.string(),
  created_at: z.string(),
  id: z.string(),
  name: z.string(),
  updated_at: z.string(),
});

export const PickupPointsSchema = z.array(PickupPointSchema);

export type TPickupPoint = z.infer<typeof PickupPointSchema>;
export type TPickupPoints = z.infer<typeof PickupPointsSchema>;
