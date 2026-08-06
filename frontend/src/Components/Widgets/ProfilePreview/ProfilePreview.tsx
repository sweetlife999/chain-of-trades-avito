import { memo } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Star } from "lucide-react";

import type { TUser } from "../../../Api/auth/auth.types";
import { logout } from "../../../Api/auth/auth";

import { useAuthDispatch } from "../../../Hooks/useAuthDispatch";
import { logoutState } from "../../../Store/authSlice";

import { Button } from "../../UI/Button/Button";

import styles from "./Styles.module.scss";
import { getAvatarGradient } from "../../Utils/getAvatarGradient";

type TProfilePreviewProps = {
  user: TUser;
};

const ProfilePreviewComponent = ({ user }: TProfilePreviewProps) => {
  const navigate = useNavigate();
  const dispatch = useAuthDispatch();

  const handleLogout = async () => {
    try {
      await logout();

      dispatch(logoutState());
      navigate("/");
    } catch (error) {
      console.log("Не удалось выйти", error);
    }
  };

  return (
    <div className={styles.profile}>
      <Link
        to={'/profile'}
        className={styles.profile__content}
      >
        <div
  className={styles.profile__avatar}
  style={{
    background: getAvatarGradient(user.id),
  }}
>
  {user.photo_url ? (
    <img
      className={styles.profile__avatarImage}
      src={user.photo_url}
      alt={user.nickname}
    />
  ) : (
    <span className={styles.profile__avatarPlaceholder}>
      {user.nickname.charAt(0).toUpperCase()}
    </span>
  )}
</div>

        <span className={styles.profile__nickname}>
          {user.nickname}
        </span>

        <div className={styles.profile__rating}>
          <span>{(user.rating ?? 1).toFixed(1)}</span>

          <Star
            className={styles.profile__star}
            fill="currentColor"
            stroke="currentColor"
          />
        </div>
      </Link>

      <div className={styles.profile__menu}>
        <div className={styles.profile__menuContent}>
          <Button
            color="transparent"
            type="button"
            onClick={handleLogout}
          >
            Выйти
          </Button>
        </div>
      </div>
    </div>
  );
};

export const ProfilePreview = memo(ProfilePreviewComponent);