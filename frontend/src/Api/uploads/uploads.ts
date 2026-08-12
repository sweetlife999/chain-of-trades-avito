import axios from "axios";

import api from "../client";
import { UploadResponseSchema } from "./uploads.types";

// Больше этого фотографию нигде не показывают: карточка обходится парой сотен пикселей,
// а галерея — шириной экрана. Телефон при этом отдаёт оригинал на 8 мегапикселей.
const maxDimension = 1600;
const quality = 0.85;

export const uploadPhoto = async (file: File): Promise<string> => {
  const form = new FormData();
  form.append("file", await downscale(file), file.name);

  const { data } = await api.post("/uploads", form);

  return UploadResponseSchema.parse(data).url;
};

// Сообщения API — английские, и показывать их человеку незачем: причин отказа ровно две,
// и обе различимы по статусу. Разбирать текст ответа не стали бы: он для разработчика.
export const getUploadErrorMessage = (
  error: unknown,
  fallback: string,
): string => {
  if (!axios.isAxiosError(error)) {
    return fallback;
  }

  switch (error.response?.status) {
    case 413:
      return "Файл больше 5 МБ";
    case 400:
      return "Нужен JPEG, PNG или WebP";
    case 401:
      return "Войдите заново и повторите";
    default:
      return fallback;
  }
};

// Уменьшаем в браузере: сервер принимает и 5 МБ, но гонять их с мобильного интернета
// незачем, а на диске они займут в двадцать раз больше нужного.
//
// Любая осечка — старый браузер, битый файл, отсутствующий webp в toBlob — возвращает
// оригинал: отправить фотографию важнее, чем сэкономить на ней трафик.
const downscale = async (file: File): Promise<Blob> => {
  try {
    const bitmap = await createImageBitmap(file);
    const scale = maxDimension / Math.max(bitmap.width, bitmap.height);

    if (scale >= 1) {
      bitmap.close();
      return file;
    }

    const canvas = document.createElement("canvas");
    canvas.width = Math.round(bitmap.width * scale);
    canvas.height = Math.round(bitmap.height * scale);

    const context = canvas.getContext("2d");
    if (!context) {
      bitmap.close();
      return file;
    }

    context.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    bitmap.close();

    const blob = await new Promise<Blob | null>((resolve) => {
      canvas.toBlob(resolve, "image/webp", quality);
    });

    return blob ?? file;
  } catch {
    return file;
  }
};
