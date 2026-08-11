import { memo } from "react";
import { Link } from "react-router-dom";

import type { TUser } from "../../../Api/auth/auth.types";

import { useLogout } from "../../../Hooks/useLogout";

import { Button } from "../../UI/Button/Button";
import { Rating } from "../../UI/Rating/Rating";

import styles from "./Styles.module.scss";
import { getAvatarGradient } from "../../Utils/getAvatarGradient";

type TProfilePreviewProps = {
  user: TUser;
};

const ProfilePreviewComponent = ({ user }: TProfilePreviewProps) => {
  const handleLogout = useLogout();

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
          <Rating size="s" value={user.rating} />
        </div>
      </Link>

      <div className={styles.profile__menu}>
        <div className={styles.profile__menuContent}>
          <Button
            className={styles.profile__logout}
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
