import { useCallback } from "react";
import { useDispatch, useSelector } from "react-redux";

import { mascotReacted, mascotReset } from "../Features/Mascot/mascotSlice";
import type { MascotEvent } from "../Features/Mascot/mascot.types";
import type { AuthDispatch, RootState } from "../Store/store";

export const useMascot = () => {
  const dispatch = useDispatch<AuthDispatch>();
  const mascot = useSelector((state: RootState) => state.mascot);

  const reactTo = useCallback(
    (event: MascotEvent) => dispatch(mascotReacted(event)),
    [dispatch],
  );
  const reset = useCallback(() => dispatch(mascotReset()), [dispatch]);

  return { ...mascot, reactTo, reset };
};
