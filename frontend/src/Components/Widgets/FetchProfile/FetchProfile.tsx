import { memo } from "react";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { Link } from "react-router-dom";
import { Button } from "../../UI/Button/Button";
import { ProfilePreview } from "../ProfilePreview/ProfilePreview";

const FetchProfileComponent = () => {
  const { isAuth, user } = useAuthSelector();

  if (isAuth && user) {
    return <ProfilePreview user={user} />;
  }

  return (
    <Link to="/login">
      <Button size="m" color="transparent">
        Вход и регистрация
      </Button>
    </Link>
  );
};

export const FetchProfile = memo(FetchProfileComponent);
