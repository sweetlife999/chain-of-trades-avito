import { memo } from "react";
import axios from "axios";
import { zodResolver } from "@hookform/resolvers/zod";
import { useFieldArray, useForm, useWatch } from "react-hook-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  getCategories,
  getItem,
  updateItem,
} from "../../../Api/items/items";
import type {
  TItem,
  TUpdateItemRequest,
} from "../../../Api/items/items.types";
import { useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { Button } from "../../UI/Button/Button";
import { Input } from "../../UI/Input/Input";
import { PhotoGallery } from "../../UI/PhotoGallery/PhotoGallery";
import {
  createChainFormSchema,
  type TCreateChainForm,
} from "../MyChainsProcess/CreateChain/shemaCreateChain";

const sameOrdered = (a: string[], b: string[]) =>
  a.length === b.length && a.every((value, index) => value === b[index]);

// Порядок фотографий важен — первая идёт в превью. Порядок желаний не важен ни здесь,
// ни на сервере, а с бэкенда они приходят отсортированными по слагу.
const sameUnordered = (a: string[], b: string[]) =>
  sameOrdered([...a].sort(), [...b].sort());

const getUpdateErrorMessage = (error: unknown) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? "Не удалось сохранить изменения";
  }

  return "Не удалось сохранить изменения";
};

type TFormProps = {
  item: TItem;
};

const ItemEditForm = ({ item }: TFormProps) => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const categoriesQuery = useQuery({
    queryKey: ["categories"],
    queryFn: getCategories,
    staleTime: 5 * 60 * 1000,
  });

  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<TCreateChainForm>({
    resolver: zodResolver(createChainFormSchema),
    defaultValues: {
      title: item.title,
      category: item.category,
      description: item.description,
      photo_urls: item.photo_urls.map((url) => ({ url })),
      wants: item.wants,
    },
  });

  const {
    fields: photoFields,
    append: appendPhoto,
    remove: removePhoto,
  } = useFieldArray({
    control,
    name: "photo_urls",
  });

  const photoValues = useWatch({
    control,
    name: "photo_urls",
  });

  const previewUrls = (photoValues ?? [])
    .map((photo) => photo.url.trim())
    .filter(Boolean);

  const updateMutation = useMutation({
    mutationFn: (request: TUpdateItemRequest) => updateItem(item.id, request),
    onSuccess: async (updatedItem) => {
      queryClient.setQueryData(["items", item.id], updatedItem);

      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["items"] }),
        queryClient.invalidateQueries({ queryKey: ["posts"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
      ]);

      navigate(`/items/${item.id}`, { replace: true });
    },
    onError: (error) => {
      setError("root.server", {
        type: "server",
        message: getUpdateErrorMessage(error),
      });
    },
  });

  // Шлём только то, что пользователь реально изменил: PATCH принимает любое подмножество полей.
  const onSubmit = (formData: TCreateChainForm) => {
    const title = formData.title.trim();
    const description = formData.description.trim();
    const photoUrls = formData.photo_urls.map(({ url }) => url.trim());

    const request: TUpdateItemRequest = {};
    if (title !== item.title) {
      request.title = title;
    }
    if (formData.category !== item.category) {
      request.category = formData.category;
    }
    if (description !== item.description) {
      request.description = description;
    }
    if (!sameOrdered(photoUrls, item.photo_urls)) {
      request.photo_urls = photoUrls;
    }
    if (!sameUnordered(formData.wants, item.wants)) {
      request.wants = formData.wants;
    }

    // Пустое тело API не примет, да и слать нечего — просто возвращаемся к карточке.
    if (Object.keys(request).length === 0) {
      navigate(`/items/${item.id}`, { replace: true });
      return;
    }

    updateMutation.mutate(request);
  };

  const categories = categoriesQuery.data ?? [];

  return (
    <main className={styles.editItem}>
      <header className={styles.editItem__header}>
        <Link className={styles.editItem__back} to={`/items/${item.id}`}>
          ← Вернуться к объявлению
        </Link>
        <h1 className={styles.editItem__title}>Редактирование объявления</h1>
        <p className={styles.editItem__description}>
          Измените данные вещи и сохраните изменения.
        </p>
      </header>

      <form
        className={styles.editItem__form}
        onSubmit={handleSubmit(onSubmit)}
        noValidate
      >
        <div className={styles.editItem__content}>
          <div className={styles.editItem__photosColumn}>
            <PhotoGallery
              urls={previewUrls}
              alt="Предпросмотр товара"
              empty={
                <div className={styles.editItem__photoPlaceholder}>
                  <span className={styles.editItem__photoPlus}>+</span>
                  <strong className={styles.editItem__photoTitle}>
                    Добавить фотографии
                  </strong>
                  <small className={styles.editItem__photoHint}>
                    Вставьте ссылки, до 10 фотографий
                  </small>
                </div>
              }
            />

            <div className={styles.editItem__photoFields}>
              {photoFields.map((field, index) => (
                <div className={styles.editItem__photoField} key={field.id}>
                  <Input
                    label={`Ссылка на фото ${index + 1}`}
                    type="text"
                    placeholder="https://example.com/photo.jpg"
                    autoComplete="url"
                    error={errors.photo_urls?.[index]?.url?.message}
                    {...register(`photo_urls.${index}.url`)}
                  />

                  {photoFields.length > 1 && (
                    <button
                      className={styles.editItem__removePhoto}
                      type="button"
                      aria-label={`Удалить фотографию ${index + 1}`}
                      onClick={() => removePhoto(index)}
                    >
                      ×
                    </button>
                  )}
                </div>
              ))}
            </div>

            <Button
              color="light"
              size="s"
              type="button"
              disabled={photoFields.length >= 10}
              onClick={() => appendPhoto({ url: "" })}
            >
              Добавить ещё фото
            </Button>
          </div>

          <div className={styles.editItem__fieldsColumn}>
            <Input
              label="Название"
              required
              type="text"
              error={errors.title?.message}
              {...register("title")}
            />

            <label className={styles.editItem__field}>
              <span className={styles.editItem__label}>
                Категория <b className={styles.editItem__required}>*</b>
              </span>

              <select
                className={`${styles.editItem__select} ${
                  errors.category ? styles.editItem__select_error : ""
                }`}
                disabled={categoriesQuery.isPending}
                {...register("category")}
              >
                <option value="">
                  {categoriesQuery.isPending
                    ? "Загрузка категорий..."
                    : "Выберите категорию"}
                </option>

                {categories.map((category) => (
                  <option key={category.slug} value={category.slug}>
                    {category.name}
                  </option>
                ))}
              </select>

              {errors.category && (
                <span className={styles.editItem__fieldError}>
                  {errors.category.message}
                </span>
              )}
            </label>

            <Input
              textarea
              label="Описание и состояние"
              required
              rows={6}
              error={errors.description?.message}
              {...register("description")}
            />

            <fieldset className={styles.editItem__wants}>
              <legend className={styles.editItem__label}>
                Что хотите получить взамен{" "}
                <b className={styles.editItem__required}>*</b>
              </legend>

              {categoriesQuery.isError ? (
                <p className={styles.editItem__fieldError}>
                  Не удалось загрузить категории
                </p>
              ) : (
                <div className={styles.editItem__categories}>
                  {categories.map((category) => (
                    <label className={styles.editItem__category} key={category.slug}>
                      <input
                        className={styles.editItem__categoryInput}
                        type="checkbox"
                        value={category.slug}
                        {...register("wants")}
                      />
                      <span className={styles.editItem__categoryLabel}>
                        {category.name}
                      </span>
                    </label>
                  ))}
                </div>
              )}

              {errors.wants && (
                <span className={styles.editItem__fieldError}>
                  {errors.wants.message}
                </span>
              )}
            </fieldset>
          </div>
        </div>

        {errors.root?.server && (
          <p className={styles.editItem__serverError}>
            {errors.root.server.message}
          </p>
        )}

        <div className={styles.editItem__actions}>
          <Button
            className={styles.editItem__action}
            color="light"
            type="button"
            disabled={updateMutation.isPending}
            onClick={() => navigate(`/items/${item.id}`)}
          >
            Отменить
          </Button>

          <Button
            className={styles.editItem__action}
            color="green"
            type="submit"
            disabled={
              updateMutation.isPending ||
              categoriesQuery.isPending ||
              categoriesQuery.isError
            }
          >
            {updateMutation.isPending ? "Сохраняем..." : "Сохранить"}
          </Button>
        </div>
      </form>
    </main>
  );
};

const ItemEditComponent = () => {
  const { id = "" } = useParams();
  const { user } = useAuthSelector();

  const itemQuery = useQuery({
    queryKey: ["items", id],
    queryFn: () => getItem(id),
    enabled: Boolean(id),
  });

  if (itemQuery.isPending) {
    return <p className={styles.editItem__state}>Загружаем объявление...</p>;
  }

  if (itemQuery.isError || !itemQuery.data) {
    return (
      <p className={styles.editItem__state_error}>
        Не удалось загрузить объявление
      </p>
    );
  }

  if (itemQuery.data.owner_id !== user?.id) {
    return (
      <div className={styles.editItem__accessError}>
        <h1 className={styles.editItem__accessTitle}>
          Редактирование недоступно
        </h1>
        <p className={styles.editItem__accessDescription}>
          Изменять можно только свои объявления.
        </p>
        <Link
          className={styles.editItem__accessLink}
          to={`/items/${itemQuery.data.id}`}
        >
          Вернуться к объявлению
        </Link>
      </div>
    );
  }

  return <ItemEditForm item={itemQuery.data} />;
};

export const ItemEdit = memo(ItemEditComponent);
