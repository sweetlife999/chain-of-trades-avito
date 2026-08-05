import { memo } from "react";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { PostsList } from "../../Widgets/PostsList/PostsList";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";

const MainComponent = () => {
  const { isAuth } = useAuthSelector();
  return (
    <>
      <div className={styles.main__titleCover}>
        <span>
          <h1 className={styles.main__title}>Цепочки</h1>
          <p className={styles.main__subtitle}>
            Обменяйте свои вещи на то, что хотите!
          </p>
        </span>
        <Button size="l" className={styles.main__button}>Обменять вещь</Button>
      </div>

      {isAuth ?<PostsList/>:
      <span className="error">Чтобы посмотреть посты необходимо зарегистрироваться</span>}
    </>
  );
};

export const Main = memo(MainComponent);
