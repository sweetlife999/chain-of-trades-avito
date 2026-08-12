import { memo } from "react";
import { z } from "zod";
import { Controller, useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";

import styles from "./Styles.module.scss";
import { updateUser } from "../../../Api/auth/auth";
import { PhotoUrlInputSchema } from "../../../Api/common.types";
import type { TAuthenticatedUser, TUser } from "../../../Api/auth/auth.types";
import {
  useAuthDispatch,
  useAuthSelector,
} from "../../../Hooks/useAuthDispatch";
import { setUserState } from "../../../Store/authSlice";
import { Button } from "../../UI/Button/Button";
import { Input } from "../../UI/Input/Input";
import { PhotoUploader } from "../../UI/PhotoUploader/PhotoUploader";

const profileEditSchema = z.object({
  nickname: z
    .string()
    .trim()
    .min(3, "Никнейм должен содержать минимум 3 символа")
    .max(32, "Никнейм должен содержать максимум 32 символа"),
  description: z.string().trim(),
  photo_url: PhotoUrlInputSchema,
});

type TProfileEdit = z.infer<typeof profileEditSchema>;

const ProfileEditComponent = () => {
  const navigate = useNavigate();
  const dispatch = useAuthDispatch();
  const queryClient = useQueryClient();
  const { user } = useAuthSelector();

  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm<TProfileEdit>({
    resolver: zodResolver(profileEditSchema),
    values: {
      nickname: user?.nickname ?? "",
      description: user?.description ?? "",
      photo_url: user?.photo_url ?? "",
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: TProfileEdit) => {
      if (!user) {
        throw new Error("Пользователь не авторизован");
      }

      return updateUser(user.id, data);
    },
    onSuccess: (updatedUser: TUser) => {
      if (!user) {
        return;
      }

      const nextUser: TAuthenticatedUser = {
        ...updatedUser,
        is_admin: user.is_admin,
      };

      dispatch(setUserState(nextUser));
      queryClient.setQueryData(["auth", "me"], nextUser);
      queryClient.setQueryData(["users", updatedUser.id], updatedUser);
      navigate("/profile");
    },
    onError: () => {
      setError("root", {
        type: "server",
        message: "Не удалось сохранить изменения",
      });
    },
  });

  if (!user) {
    return <p className={styles.edit__message}>Пользователь не авторизован</p>;
  }

  return (
    <section className={styles.edit}>
      <div className={styles.edit__header}>
        <h1 className={styles.edit__title}>Редактирование профиля</h1>
        <p className={styles.edit__description}>
          Измените данные, которые будут видны другим пользователям.
        </p>
      </div>

      <form
        className={styles.edit__form}
        noValidate
        onSubmit={handleSubmit((data) => updateMutation.mutate(data))}
      >
        <Input
          label="Nickname"
          required
          error={errors.nickname?.message}
          {...register("nickname")}
        />

        <div className={styles.edit__field}>
          <span className={styles.edit__label}>Фотография профиля</span>

          {/* Пустая строка — «без фотографии», ровно то, чем её кодирует API. */}
          <Controller
            control={control}
            name="photo_url"
            render={({ field }) => (
              <PhotoUploader
                max={1}
                urls={field.value ? [field.value] : []}
                onChange={(urls) => field.onChange(urls[0] ?? "")}
                disabled={updateMutation.isPending}
              />
            )}
          />

          {errors.photo_url && (
            <span className={styles.edit__error}>
              {errors.photo_url.message}
            </span>
          )}
        </div>

        <Input
          textarea
          label="Описание"
          rows={5}
          error={errors.description?.message}
          {...register("description")}
        />

        {errors.root && (
          <p className={styles.edit__error}>{errors.root.message}</p>
        )}

        <div className={styles.edit__actions}>
          <Button
            className={styles.edit__action}
            color="transparent"
            disabled={updateMutation.isPending}
            type="button"
            onClick={() => navigate("/profile")}
          >
            Отменить
          </Button>
          <Button
            className={styles.edit__action}
            disabled={updateMutation.isPending}
            type="submit"
          >
            {updateMutation.isPending ? "Сохраняем..." : "Сохранить"}
          </Button>
        </div>
      </form>
    </section>
  );
};

export const ProfileEdit = memo(ProfileEditComponent);
