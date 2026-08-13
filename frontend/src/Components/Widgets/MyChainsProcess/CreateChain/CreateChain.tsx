import { memo } from "react";
import axios from "axios";
import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import styles from "./Styles.module.scss";
import { createItem, getCategories } from "../../../../Api/items/items";
import {
  createChainFormSchema,
  type TCreateChainForm,
} from "./shemaCreateChain";
import type { TCreateItemRequest } from "../../../../Api/items/items.types";
import { Input } from "../../../UI/Input/Input";
import { Button } from "../../../UI/Button/Button";
import { PhotoUploader } from "../../../UI/PhotoUploader/PhotoUploader";

const getRequestErrorMessage = (error: unknown) => {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? "Не удалось создать объявление";
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
      photo_urls: [],
      wants: [],
      search_filters: {
        max_chain_length: 5,
        min_participant_rating: 0,
        prefer_reliable_participants: true,
      },
    },
  });

  const createItemMutation = useMutation({
    mutationFn: (request: TCreateItemRequest) => createItem(request),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["posts"] }),
        queryClient.invalidateQueries({ queryKey: ["items"] }),
        queryClient.invalidateQueries({ queryKey: ["exchanges"] }),
      ]);

      navigate("/myItems");
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
      photo_urls: formData.photo_urls,
      wants: formData.wants,
      search_filters: formData.search_filters,
    });
  };

  const categories = categoriesQuery.data ?? [];

  return (
    <div className={styles.createChain}>
      <div className={styles.createChain__header}>
        <h1 className={styles.createChain__title}>Создать цепочку</h1>

        <p className={styles.createChain__subtitle}>
          Опишите товар и то, что хотите получить взамен
        </p>
      </div>

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
              <span className={styles.createChain__label}>
                Фотографии <b className={styles.createChain__required}>*</b>
              </span>

              <Controller
                control={control}
                name="photo_urls"
                render={({ field }) => (
                  <PhotoUploader
                    urls={field.value}
                    onChange={field.onChange}
                    disabled={createItemMutation.isPending}
                  />
                )}
              />

              {errors.photo_urls && (
                <span className={styles.createChain__fieldError}>
                  {errors.photo_urls.message}
                </span>
              )}
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
                  Категория <b className={styles.createChain__required}>*</b>
                </span>

                <select
                  className={`${styles.createChain__select} ${
                    errors.category ? styles.createChain__select_error : ""
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
                  Что хотите получить взамен{" "}
                  <b className={styles.createChain__required}>*</b>
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
                          className={styles.createChain__categoryInput}
                          type="checkbox"
                          value={category.slug}
                          {...register("wants")}
                        />

                        <span className={styles.createChain__categoryLabel}>
                          {category.name}
                        </span>
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

              <fieldset className={styles.createChain__wants}>
                <legend className={styles.createChain__label}>
                  Настройки поиска
                </legend>

                <label className={styles.createChain__field}>
                  <span className={styles.createChain__label}>
                    Максимальная длина цепочки
                  </span>
                  <select
                    className={styles.createChain__select}
                    disabled={createItemMutation.isPending}
                    {...register("search_filters.max_chain_length", {
                      valueAsNumber: true,
                    })}
                  >
                    <option value={2}>2 участника</option>
                    <option value={3}>До 3 участников</option>
                    <option value={4}>До 4 участников</option>
                    <option value={5}>До 5 участников</option>
                  </select>
                </label>

                <label className={styles.createChain__field}>
                  <span className={styles.createChain__label}>
                    Минимальный рейтинг участников
                  </span>
                  <select
                    className={styles.createChain__select}
                    disabled={createItemMutation.isPending}
                    {...register("search_filters.min_participant_rating", {
                      valueAsNumber: true,
                    })}
                  >
                    <option value={0}>Не учитывать рейтинг</option>
                    <option value={3}>От 3.0</option>
                    <option value={3.5}>От 3.5</option>
                    <option value={4}>От 4.0</option>
                    <option value={4.5}>От 4.5</option>
                    <option value={5}>Только 5.0</option>
                  </select>
                </label>

                <label className={styles.createChain__category}>
                  <input
                    className={styles.createChain__categoryInput}
                    disabled={createItemMutation.isPending}
                    type="checkbox"
                    {...register(
                      "search_filters.prefer_reliable_participants",
                    )}
                  />
                  <span className={styles.createChain__categoryLabel}>
                    Сначала показывать обмены с надёжными участниками
                  </span>
                </label>
              </fieldset>
            </div>
          </div>

          {errors.root?.server && (
            <p className={styles.createChain__serverError} role="alert">
              {errors.root.server.message}
            </p>
          )}

          <div className={styles.createChain__actions}>
            <Button
              className={styles.createChain__action}
              color="transparent"
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
              {createItemMutation.isPending ? "Создание..." : "Продолжить"}
            </Button>
          </div>
        </form>
      </section>
    </div>
  );
};

export const CreateChain = memo(CreateChainComponent);
