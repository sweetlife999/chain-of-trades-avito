import { memo } from "react";
import styles from "./Styles.module.scss";
import { Button } from "../../UI/Button/Button";
import { PostsList } from "../../Widgets/PostsList/PostsList";
import { Post } from "../../Widgets/Post/Post";

const MainComponent = () => {
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

      <Post/>
      <PostsList/>
    </>
  );
};

export const Main = memo(MainComponent);
