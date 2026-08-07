import type { ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { Loader } from "lucide-react";

type QueryStatus = UseQueryResult<unknown, unknown>["status"];

interface FetchStatusProps {
  status: QueryStatus;
  children: ReactNode;
  loadingFallback?: ReactNode;
  errorFallback?: ReactNode;
}

export const FetchStatus = ({
  status,
  children,
  loadingFallback = <Loader />,
  errorFallback = <span>Произошла ошибка</span>,
}: FetchStatusProps) => {
  switch (status) {
    case "pending":
      return <>{loadingFallback}</>;

    case "error":
      return <>{errorFallback}</>;

    case "success":
      return <>{children}</>;
  }
};