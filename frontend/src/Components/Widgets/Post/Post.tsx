import { memo } from "react";
import styles from "./Styles.module.scss";
import arrow from "/src/Assets/thick-arrow.svg";

const PostComponent = () => {
  const data = {
    id: 4,
    img1: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQxGUUrdmZZJuQDy2wOMi_U4K6w52csKmy4UgFBwqdc4A&s=10",
    img2: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTbOAycTdIR-7yrPHNAmnRYQedvxPgdsW1aGrKfipViTQ&s=10",
    thing1: "Банан",
    thing2: "Пирог",
    status: "Подтверждение",
    countUsers: 3,
  };

  const gradients = [
    "linear-gradient(135deg, aqua, #6a11cb)",
    "linear-gradient(135deg, #ff6b6b, #feca57)",
    "linear-gradient(135deg, #48c6ef, #6f86d6)",
    "linear-gradient(135deg, #f093fb, #f5576c)",
    "linear-gradient(135deg, #4facfe, #00f2fe)",
  ];

  return (
    <article className={styles.post}>
      <div
        className={styles.post__imgs}
        style={{ background: gradients[data.id % gradients.length] }}
      >
        <img
          className={styles.post__img}
          src={data.img1}
          alt={data.thing1}
        />
        <img
          className={`${styles.post__img} ${styles.post__arrow}`}
          src={arrow}
        />
        <img
          className={styles.post__img}
          src={data.img2}
          alt={data.thing2}
        />
      </div>
      <h3 className={styles.post__trade}>
        {data.thing1} → {data.thing2}
      </h3>
      <p className={styles.post__users}>
        Участников: {data.countUsers}
      </p>
    </article>
  );
};

export const Post = memo(PostComponent);