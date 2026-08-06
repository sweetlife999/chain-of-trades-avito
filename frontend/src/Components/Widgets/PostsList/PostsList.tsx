import { memo } from "react";
import { useQuery } from "@tanstack/react-query";

import styles from "./Styles.module.scss";
import { getItems } from "../../../Api/items/items";
import { FetchStatus } from "../FetchStatus/FetchStatus";
import { Post } from "../Post/Post";

const PostsListComponent = () => {
  const itemsQuery = useQuery({
    queryKey: ["items"],
    queryFn: getItems,
  });

  return (
    <FetchStatus status={itemsQuery.status}>
      <ul className={styles.posts}>
        {itemsQuery.data?.map((item) => (
          <li key={item.id}>
            <Post post={item} />
          </li>
        ))}
      </ul>
    </FetchStatus>
  );
};

export const PostsList = memo(PostsListComponent);