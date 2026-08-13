import { useQuery } from "@tanstack/react-query";

import { getCategories } from "../Api/items/items";

// API работает слагами — они стабильный ключ, а название меняется. Русское имя
// живёт только в справочнике, поэтому подставляем его на экране, а не в запросе.
export const useCategoryName = () => {
  const { data } = useQuery({ queryKey: ["categories"], queryFn: getCategories });
  const bySlug = new Map(data?.map((category) => [category.slug, category.name]));

  // Пока справочник грузится (или в нём нет слага) показываем сам слаг: это
  // хуже названия, но лучше пустого места на карточке.
  return (slug: string) => bySlug.get(slug) ?? slug;
};
