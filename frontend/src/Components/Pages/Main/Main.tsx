import { memo } from "react";
import { useNavigate } from "react-router-dom";

import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { PostsList } from "../../Widgets/PostsList/PostsList";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";

const MainComponent = () => {
  const navigate = useNavigate();
  const { isAuth } = useAuthSelector();

  return (
    <section>
      <header className={styles.main__titleCover}>
        <div>
          <h1 className={styles.main__title}>Обмены</h1>
          <p className={styles.main__subtitle}>Найдите подходящую цепочку обмена</p>
        </div>
        <Button size="l" onClick={() => navigate("/exchanges/create")}>
          Добавить вещь
        </Button>
      </header>

      {isAuth ? (
        <PostsList />
      ) : (
        <p className="error">Чтобы посмотреть обмены, войдите в аккаунт</p>
      )}
    </section>
  );
};

export const Main = memo(MainComponent);
