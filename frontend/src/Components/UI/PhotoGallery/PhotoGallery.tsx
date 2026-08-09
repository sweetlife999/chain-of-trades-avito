import {
  memo,
  useEffect,
  useState,
  type MouseEvent,
  type ReactNode,
} from "react";

import styles from "./Styles.module.scss";

type TProps = {
  urls: string[];
  alt: string;
  empty?: ReactNode;
};

const PhotoGalleryComponent = ({ urls, alt, empty }: TProps) => {
  const [activeIndex, setActiveIndex] = useState(0);
  const [viewerIndex, setViewerIndex] = useState<number | null>(null);

  const safeActiveIndex = urls.length
    ? Math.min(activeIndex, urls.length - 1)
    : 0;
  const safeViewerIndex =
    viewerIndex === null || !urls.length
      ? null
      : Math.min(viewerIndex, urls.length - 1);
  const hasMultiplePhotos = urls.length > 1;

  useEffect(() => {
    if (safeViewerIndex === null) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setViewerIndex(null);
        return;
      }

      if (!hasMultiplePhotos) {
        return;
      }

      if (event.key === "ArrowLeft") {
        setViewerIndex((current) => {
          const index = current ?? 0;
          return (index - 1 + urls.length) % urls.length;
        });
      }

      if (event.key === "ArrowRight") {
        setViewerIndex((current) => {
          const index = current ?? 0;
          return (index + 1) % urls.length;
        });
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [hasMultiplePhotos, safeViewerIndex, urls.length]);

  const showPrevious = () => {
    setActiveIndex(
      (safeActiveIndex - 1 + urls.length) % urls.length,
    );
  };

  const showNext = () => {
    setActiveIndex((safeActiveIndex + 1) % urls.length);
  };

  const showPreviousInViewer = () => {
    if (safeViewerIndex === null) {
      return;
    }

    setViewerIndex((safeViewerIndex - 1 + urls.length) % urls.length);
  };

  const showNextInViewer = () => {
    if (safeViewerIndex === null) {
      return;
    }

    setViewerIndex((safeViewerIndex + 1) % urls.length);
  };

  const handleViewerOverlayClick = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      setViewerIndex(null);
    }
  };

  if (!urls.length) {
    return <div className={styles.gallery__empty}>{empty}</div>;
  }

  return (
    <>
      <div className={styles.gallery}>
        <button
          className={styles.gallery__imageButton}
          type="button"
          aria-label={`Открыть фотографию ${safeActiveIndex + 1} из ${urls.length}`}
          onClick={() => setViewerIndex(safeActiveIndex)}
        >
          <img
            className={styles.gallery__image}
            src={urls[safeActiveIndex]}
            alt={`${alt}, фото ${safeActiveIndex + 1}`}
            draggable={false}
          />
        </button>

        {hasMultiplePhotos && (
          <>
            <button
              className={`${styles.gallery__arrow} ${styles.gallery__arrow_left}`}
              type="button"
              aria-label="Предыдущая фотография"
              onClick={showPrevious}
            >
              ‹
            </button>
            <button
              className={`${styles.gallery__arrow} ${styles.gallery__arrow_right}`}
              type="button"
              aria-label="Следующая фотография"
              onClick={showNext}
            >
              ›
            </button>

            <span className={styles.gallery__counter}>
              {safeActiveIndex + 1} / {urls.length}
            </span>
          </>
        )}
      </div>

      {safeViewerIndex !== null && (
        <div
          className={styles.viewer}
          role="dialog"
          aria-modal="true"
          aria-label="Просмотр фотографии"
          onClick={handleViewerOverlayClick}
        >
          <button
            className={styles.viewer__close}
            type="button"
            aria-label="Закрыть просмотр"
            onClick={() => setViewerIndex(null)}
          >
            ×
          </button>

          {hasMultiplePhotos && (
            <button
              className={`${styles.viewer__arrow} ${styles.viewer__arrow_left}`}
              type="button"
              aria-label="Предыдущая фотография"
              onClick={showPreviousInViewer}
            >
              ‹
            </button>
          )}

          <img
            className={styles.viewer__image}
            src={urls[safeViewerIndex]}
            alt={`${alt}, фото ${safeViewerIndex + 1}`}
          />

          {hasMultiplePhotos && (
            <>
              <button
                className={`${styles.viewer__arrow} ${styles.viewer__arrow_right}`}
                type="button"
                aria-label="Следующая фотография"
                onClick={showNextInViewer}
              >
                ›
              </button>
              <span className={styles.viewer__counter}>
                {safeViewerIndex + 1} / {urls.length}
              </span>
            </>
          )}
        </div>
      )}
    </>
  );
};

export const PhotoGallery = memo(PhotoGalleryComponent);
