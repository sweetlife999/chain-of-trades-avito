import { memo } from "react";
import styles from "./Styles.module.scss";
import { PostsList } from "../../Widgets/PostsList/PostsList";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { AuthRequiredState } from "../../UI/AuthRequiredState/AuthRequiredState";

const MainComponent = () => {
  const { isAuth } = useAuthSelector();

  return (
    <section className={styles.main}>
      <div className={styles.main__titleCover}>
        <div className={styles.main__heading}>
          <h1 className={styles.main__title}>Обмены</h1>
          <p className={styles.main__subtitle}>
            Найдите подходящую цепочку обмена
          </p>
        </div>
      </div>

      {isAuth ? (
        <PostsList />
      ) : (
        <AuthRequiredState
          description="После входа вы увидите найденные цепочки, сможете добавить свою вещь и отслеживать поиск новых вариантов обмена."
          returnTo="/feed"
          title="Войдите, чтобы посмотреть обмены"
        />
      )}
    </section>
  );
};

export const Main = memo(MainComponent);
