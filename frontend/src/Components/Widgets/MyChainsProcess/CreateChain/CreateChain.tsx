import { memo } from "react";
import axios from "axios";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  useFieldArray,
  useForm,
  useWatch,
} from "react-hook-form";
import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import styles from "./Styles.module.scss";
import { createItem, getCategories } from "../../../../Api/items/items";
import { createChainFormSchema, type TCreateChainForm } from "./shemaCreateChain";
import type { TCreateItemRequest } from "../../../../Api/items/items.types";
import { Input } from "../../../UI/Input/Input";
import { Button } from "../../../UI/Button/Button";



const getRequestErrorMessage = (error: unknown) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return (
      error.response?.data?.error ??
      "Не удалось создать объявление"
    );
  }

  return "Не удалось создать объявление";
};

const CreateChainComponent = () => {
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
      title: "",
      category: "",
      description: "",
      photo_urls: [{ url: "" }],
      wants: [],
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

  const previewUrl = photoValues?.find(
    (photo) => photo.url.trim().length > 0,
  )?.url;

  const createItemMutation = useMutation({
    mutationFn: (request: TCreateItemRequest) =>
      createItem(request),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["posts"] }),
        queryClient.invalidateQueries({ queryKey: ["items"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
      ]);

      navigate("/myChains");
    },
    onError: (error) => {
      setError("root.server", {
        type: "server",
        message: getRequestErrorMessage(error),
      });
    },
  });

  const onSubmit = (formData: TCreateChainForm) => {
    createItemMutation.mutate({
      title: formData.title.trim(),
      category: formData.category,
      description: formData.description.trim(),
      photo_urls: formData.photo_urls.map(({ url }) =>
        url.trim(),
      ),
      wants: formData.wants,
    });
  };

  const categories = categoriesQuery.data ?? [];

  return (
    <main className={styles.createChain}>
      <header className={styles.createChain__header}>
        <h1 className={styles.createChain__title}>
          Создать цепочку
        </h1>

        <p className={styles.createChain__subtitle}>
          Опишите товар и то, что хотите получить взамен
        </p>
      </header>

      <section className={styles.createChain__section}>
        <h2 className={styles.createChain__sectionTitle}>
          Расскажите о вашей вещи
        </h2>

        <form
          className={styles.createChain__form}
          onSubmit={handleSubmit(onSubmit)}
          noValidate
        >
          <div className={styles.createChain__content}>
            <div className={styles.createChain__photosColumn}>
              <div className={styles.createChain__photoPreview}>
                {previewUrl ? (
                  <img
                    className={styles.createChain__photoImage}
                    src={previewUrl}
                    alt="Предпросмотр товара"
                  />
                ) : (
                  <div className={styles.createChain__photoPlaceholder}>
                    <span className={styles.createChain__photoPlus}>
                      +
                    </span>

                    <strong>Добавить фотографии</strong>

                    <small>
                      Вставьте ссылки, до 10 фотографий
                    </small>
                  </div>
                )}
              </div>

              <div className={styles.createChain__photoFields}>
                {photoFields.map((field, index) => (
                  <div
                    className={styles.createChain__photoField}
                    key={field.id}
                  >
                    <Input
                      label={`Ссылка на фото ${index + 1}`}
                      type="text"
                      placeholder="https://example.com/photo.jpg"
                      autoComplete="url"
                      error={
                        errors.photo_urls?.[index]?.url?.message
                      }
                      {...register(`photo_urls.${index}.url`)}
                    />

                    {photoFields.length > 1 && (
                      <button
                        className={styles.createChain__removePhoto}
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
                className={styles.createChain__addPhoto}
                color="light"
                size="s"
                type="button"
                disabled={photoFields.length >= 10}
                onClick={() => appendPhoto({ url: "" })}
              >
                Добавить ещё фото
              </Button>
            </div>

            <div className={styles.createChain__fieldsColumn}>
              <Input
                label="Название"
                required
                type="text"
                placeholder="Например, велосипед Trek 820"
                error={errors.title?.message}
                {...register("title")}
              />

              <label className={styles.createChain__field}>
                <span className={styles.createChain__label}>
                  Категория <b>*</b>
                </span>

                <select
                  className={`${styles.createChain__select} ${
                    errors.category
                      ? styles.createChain__select_error
                      : ""
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
                    <option
                      key={category.slug}
                      value={category.slug}
                    >
                      {category.name}
                    </option>
                  ))}
                </select>

                {errors.category && (
                  <span className={styles.createChain__fieldError}>
                    {errors.category.message}
                  </span>
                )}
              </label>

              <Input
                textarea
                label="Описание и состояние"
                required
                rows={6}
                placeholder="Укажите состояние, царапины, сколы, комплектацию и другие важные детали"
                error={errors.description?.message}
                {...register("description")}
              />

              <fieldset className={styles.createChain__wants}>
                <legend className={styles.createChain__label}>
                  Что хотите получить взамен <b>*</b>
                </legend>

                {categoriesQuery.isError ? (
                  <p className={styles.createChain__fieldError}>
                    Не удалось загрузить категории
                  </p>
                ) : (
                  <div className={styles.createChain__categories}>
                    {categories.map((category) => (
                      <label
                        className={styles.createChain__category}
                        key={category.slug}
                      >
                        <input
                          type="checkbox"
                          value={category.slug}
                          {...register("wants")}
                        />

                        <span>{category.name}</span>
                      </label>
                    ))}
                  </div>
                )}

                {errors.wants && (
                  <span className={styles.createChain__fieldError}>
                    {errors.wants.message}
                  </span>
                )}
              </fieldset>
            </div>
          </div>

          {errors.root?.server && (
            <p className={styles.createChain__serverError}>
              {errors.root.server.message}
            </p>
          )}

          <div className={styles.createChain__actions}>
            <Button
              className={styles.createChain__action}
              color="green"
              type="button"
              disabled={createItemMutation.isPending}
              onClick={() => navigate(-1)}
            >
              Отменить
            </Button>

            <Button
              className={styles.createChain__action}
              color="green"
              type="submit"
              disabled={
                createItemMutation.isPending ||
                categoriesQuery.isPending ||
                categoriesQuery.isError
              }
            >
              {createItemMutation.isPending
                ? "Создание..."
                : "Продолжить"}
            </Button>
          </div>
        </form>
      </section>
    </main>
  );
};

export const CreateChain = memo(CreateChainComponent);