import { memo, useRef, useState, type ChangeEvent } from "react";

import styles from "./Styles.module.scss";
import {
  getUploadErrorMessage,
  uploadPhoto,
} from "../../../Api/uploads/uploads";

type TProps = {
  urls: string[];
  onChange: (urls: string[]) => void;
  max?: number;
  disabled?: boolean;
};

type TQueued = {
  id: number;
  name: string;
  error?: string;
};

const PhotoUploaderComponent = ({
  urls,
  onChange,
  max = 10,
  disabled = false,
}: TProps) => {
  const inputRef = useRef<HTMLInputElement>(null);
  const nextId = useRef(0);
  const [queue, setQueue] = useState<TQueued[]>([]);

  // Порядок имеет смысл только там, где фотографий несколько: у аватарки обложки нет.
  const ordered = max > 1;
  // С одной фотографией новая заменяет старую — на аватарке это единственное, чего от
  // формы ждут. Со списком место кончается, и кнопка добавления пропадает.
  const room = ordered ? max - urls.length : 1;
  const uploading = queue.filter((item) => !item.error).length;
  // Кнопка называется тем, что произойдёт: с аватаркой файл заменяется, а не добавляется.
  const addLabel = (() => {
    if (urls.length === 0) {
      return "Добавить фотографии";
    }

    return ordered ? "Добавить ещё" : "Заменить фотографию";
  })();

  const handleSelect = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []).slice(0, room);
    // Тот же файл после ошибки выбирают повторно, а без сброса значения change на нём
    // второй раз не срабатывает.
    event.target.value = "";

    if (files.length === 0) {
      return;
    }

    const queued = files.map((file) => ({ id: nextId.current++, name: file.name }));
    setQueue((current) => [...current, ...queued]);

    // Последовательно и с локальным накопителем: параллельные загрузки дописывали бы
    // ссылку каждая к своему снимку urls, и до формы доехала бы только последняя.
    let next = urls;
    for (const [index, file] of files.entries()) {
      const { id } = queued[index];

      try {
        next = [...next, await uploadPhoto(file)].slice(-max);
        onChange(next);
        setQueue((current) => current.filter((item) => item.id !== id));
      } catch (error) {
        const message = getUploadErrorMessage(error, "Не удалось загрузить");
        setQueue((current) =>
          current.map((item) => (item.id === id ? { ...item, error: message } : item)),
        );
      }
    }
  };

  const remove = (index: number) => {
    onChange(urls.filter((_, position) => position !== index));
  };

  const move = (index: number, direction: -1 | 1) => {
    const next = [...urls];
    [next[index], next[index + direction]] = [next[index + direction], next[index]];
    onChange(next);
  };

  return (
    <div className={styles.uploader}>
      <ul className={styles.uploader__grid}>
        {urls.map((url, index) => (
          <li className={styles.uploader__tile} key={`${index}-${url}`}>
            <img
              className={styles.uploader__image}
              src={url}
              alt={`Фотография ${index + 1}`}
            />

            {ordered && index === 0 && (
              <span className={styles.uploader__cover}>Обложка</span>
            )}

            <button
              className={styles.uploader__remove}
              type="button"
              disabled={disabled}
              aria-label={`Удалить фотографию ${index + 1}`}
              onClick={() => remove(index)}
            >
              ×
            </button>

            {ordered && urls.length > 1 && (
              <div className={styles.uploader__move}>
                <button
                  className={styles.uploader__moveButton}
                  type="button"
                  disabled={disabled || index === 0}
                  aria-label={`Переставить фотографию ${index + 1} назад`}
                  onClick={() => move(index, -1)}
                >
                  ←
                </button>

                <button
                  className={styles.uploader__moveButton}
                  type="button"
                  disabled={disabled || index === urls.length - 1}
                  aria-label={`Переставить фотографию ${index + 1} вперёд`}
                  onClick={() => move(index, 1)}
                >
                  →
                </button>
              </div>
            )}
          </li>
        ))}

        {queue.map((item) => (
          <li
            className={`${styles.uploader__tile} ${
              item.error ? styles.uploader__tile_failed : ""
            }`}
            key={item.id}
          >
            {item.error ? (
              <div className={styles.uploader__failure}>
                <strong className={styles.uploader__failureName}>{item.name}</strong>
                <span className={styles.uploader__failureMessage}>{item.error}</span>

                <button
                  className={styles.uploader__remove}
                  type="button"
                  aria-label={`Убрать ${item.name} из списка`}
                  onClick={() =>
                    setQueue((current) =>
                      current.filter((queued) => queued.id !== item.id),
                    )
                  }
                >
                  ×
                </button>
              </div>
            ) : (
              <div className={styles.uploader__progress}>
                <span className={styles.uploader__spinner} />
              </div>
            )}
          </li>
        ))}

        {room > 0 && (
          <li className={styles.uploader__tile_empty}>
            <button
              className={styles.uploader__add}
              type="button"
              disabled={disabled}
              onClick={() => inputRef.current?.click()}
            >
              <span className={styles.uploader__addSign} aria-hidden="true">
                +
              </span>
              {addLabel}
            </button>
          </li>
        )}
      </ul>

      <input
        className={styles.uploader__input}
        ref={inputRef}
        type="file"
        multiple={ordered}
        accept="image/jpeg,image/png,image/webp"
        tabIndex={-1}
        onChange={handleSelect}
      />

      <p className={styles.uploader__hint}>
        {ordered
          ? `JPEG, PNG или WebP, до ${max} фотографий. Первая станет обложкой объявления.`
          : "JPEG, PNG или WebP. Новая фотография заменит текущую."}
      </p>

      <span className={styles.uploader__status} role="status">
        {uploading > 0 ? `Загружаем фотографии: ${uploading}` : ""}
      </span>
    </div>
  );
};

export const PhotoUploader = memo(PhotoUploaderComponent);
