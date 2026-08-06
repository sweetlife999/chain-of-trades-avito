import { memo } from "react";
import { Post } from "../Post/Post";
import { useQuery } from "@tanstack/react-query";
import { getItems } from "../../../Api/items/items";
import { FetchStatus } from "../FetchStatus/FetchStatus";

const PostsListComponent = () => {
  const postsList = useQuery({
    queryFn: () => getItems(),
    queryKey: ["posts"],
  });

  

  return (
    <FetchStatus status={postsList.status}>
      <ul>
         {postsList.data?.map((item) => (
          <li key={item.id}>
            <Post post={item} />
          </li>
        ))}
      </ul>
    </FetchStatus>
  );
};

export const PostsList = memo(PostsListComponent);
