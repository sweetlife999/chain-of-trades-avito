import { Navigate } from "react-router-dom";

import { useAuthSelector } from "../../../../Hooks/useAuthDispatch";
import { Admin } from "../Admin/Admin";

export const AdminRoute = () => {
  const { isAdmin, isAuth } = useAuthSelector();

  if (!isAuth) {
    return <Navigate state={{ from: "/admin" }} to="/login" replace />;
  }

  if (!isAdmin) {
    return <Navigate to="/feed" replace />;
  }

  return <Admin />;
};
