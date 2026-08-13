import {
  memo,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import clsx from "clsx";
import { useSelector } from "react-redux";
import { useLocation } from "react-router-dom";

import styles from "./Styles.module.scss";
import { shouldShowMascotGuide } from "../../../Features/Mascot/mascotVisibility";
import { useMascot } from "../../../Hooks/useMascot";
import { Mascot } from "../Mascot/Mascot";
import type { RootState } from "../../../Store/store";

const viewportGap = 12;
const contentGap = 8;
const headerGap = 10;
const targetGap = 14;

type TPoint = {
  x: number;
  y: number;
};

type TPosition = TPoint & {
  animate: boolean;
  visible: boolean;
};

const clamp = (value: number, min: number, max: number) =>
  Math.min(Math.max(value, min), Math.max(min, max));

const intersects = (first: DOMRect, second: DOMRect) =>
  first.left < second.right &&
  first.right > second.left &&
  first.top < second.bottom &&
  first.bottom > second.top;

const expandRect = (rect: DOMRect, gap: number) =>
  new DOMRect(
    rect.left - gap,
    rect.top - gap,
    rect.width + gap * 2,
    rect.height + gap * 2,
  );

const isVisibleRect = (rect: DOMRect, width: number, height: number) =>
  rect.width > 0 &&
  rect.height > 0 &&
  rect.right > 0 &&
  rect.bottom > 0 &&
  rect.left < width &&
  rect.top < height;

const collectProtectedRects = (
  guide: HTMLElement,
  width: number,
  height: number,
) => {
  const protectedRects: DOMRect[] = [];
  const addRect = (rect: DOMRect, gap = contentGap) => {
    if (isVisibleRect(rect, width, height)) {
      protectedRects.push(expandRect(rect, gap));
    }
  };

  const walker = document.createTreeWalker(
    document.body,
    NodeFilter.SHOW_TEXT,
    {
      acceptNode: (node) => {
        if (
          !node.textContent?.trim() ||
          !node.parentElement ||
          guide.contains(node)
        ) {
          return NodeFilter.FILTER_REJECT;
        }

        const style = window.getComputedStyle(node.parentElement);
        return style.display === "none" || style.visibility === "hidden"
          ? NodeFilter.FILTER_REJECT
          : NodeFilter.FILTER_ACCEPT;
      },
    },
  );

  let textNode = walker.nextNode();
  while (textNode) {
    const range = document.createRange();
    range.selectNodeContents(textNode);
    Array.from(range.getClientRects()).forEach((rect) => addRect(rect));
    range.detach();
    textNode = walker.nextNode();
  }

  document
    .querySelectorAll<HTMLElement>(
      [
        "button",
        "input",
        "textarea",
        "select",
        "a",
        "img",
        "video",
        "canvas",
        "article",
        "li",
        "form",
        "table",
        "[role='dialog']",
        "[data-mascot-protected]",
      ].join(","),
    )
    .forEach((element) => {
      if (!guide.contains(element)) {
        addRect(element.getBoundingClientRect());
      }
    });

  const pageHeader = document.querySelector<HTMLElement>("body > * header");
  if (pageHeader && !guide.contains(pageHeader)) {
    addRect(pageHeader.getBoundingClientRect(), headerGap);
  }

  return protectedRects;
};

const createGuideRect = (
  point: TPoint,
  guideWidth: number,
  guideHeight: number,
) => new DOMRect(point.x, point.y, guideWidth, guideHeight);

const isSafePoint = (
  point: TPoint,
  guideWidth: number,
  guideHeight: number,
  protectedRects: DOMRect[],
) => {
  const guideRect = createGuideRect(point, guideWidth, guideHeight);
  return !protectedRects.some((rect) => intersects(guideRect, rect));
};

const isSafePath = (
  from: TPoint,
  to: TPoint,
  guideWidth: number,
  guideHeight: number,
  protectedRects: DOMRect[],
) => {
  const distance = Math.hypot(to.x - from.x, to.y - from.y);
  const steps = Math.max(1, Math.ceil(distance / 24));

  for (let step = 1; step <= steps; step += 1) {
    const progress = step / steps;
    const point = {
      x: Math.round(from.x + (to.x - from.x) * progress),
      y: Math.round(from.y + (to.y - from.y) * progress),
    };

    if (!isSafePoint(point, guideWidth, guideHeight, protectedRects)) {
      return false;
    }
  }

  return true;
};

const getPreferredPoint = (
  target: HTMLElement | null,
  guideWidth: number,
  guideHeight: number,
  width: number,
  height: number,
) => {
  const maxX = Math.max(viewportGap, width - guideWidth - viewportGap);
  const maxY = Math.max(viewportGap, height - guideHeight - viewportGap);

  if (!target) {
    return { x: maxX, y: maxY };
  }

  const rect = target.getBoundingClientRect();
  return {
    x: clamp(rect.right + targetGap, viewportGap, maxX),
    y: clamp(
      rect.top + rect.height / 2 - guideHeight / 2,
      viewportGap,
      maxY,
    ),
  };
};

const createCandidates = (
  preferred: TPoint,
  target: HTMLElement | null,
  guideWidth: number,
  guideHeight: number,
  width: number,
  height: number,
) => {
  const maxX = Math.max(viewportGap, width - guideWidth - viewportGap);
  const maxY = Math.max(viewportGap, height - guideHeight - viewportGap);
  const candidates: TPoint[] = [preferred];

  if (target) {
    const rect = target.getBoundingClientRect();
    const targetCandidates = [
      {
        x: rect.left - guideWidth - targetGap,
        y: rect.top + rect.height / 2 - guideHeight / 2,
      },
      {
        x: rect.left + rect.width / 2 - guideWidth / 2,
        y: rect.top - guideHeight - targetGap,
      },
      {
        x: rect.left + rect.width / 2 - guideWidth / 2,
        y: rect.bottom + targetGap,
      },
    ];

    targetCandidates.forEach((point) => {
      candidates.push({
        x: clamp(point.x, viewportGap, maxX),
        y: clamp(point.y, viewportGap, maxY),
      });
    });
  }

  for (let y = viewportGap; y <= maxY; y += 36) {
    for (let x = viewportGap; x <= maxX; x += 36) {
      candidates.push({ x, y });
    }
  }

  candidates.push(
    { x: viewportGap, y: viewportGap },
    { x: maxX, y: viewportGap },
    { x: viewportGap, y: maxY },
    { x: maxX, y: maxY },
  );

  return candidates.sort(
    (first, second) =>
      Math.hypot(first.x - preferred.x, first.y - preferred.y) -
      Math.hypot(second.x - preferred.x, second.y - preferred.y),
  );
};

const MascotGuideComponent = () => {
  const { pathname } = useLocation();
  const { anchor, message, movement, reset, revision } = useMascot();
  const guideRef = useRef<HTMLElement>(null);
  const previousPathRef = useRef(pathname);
  const [roamStep, setRoamStep] = useState(0);
  const tutorialActive = useSelector(
    (state: RootState) => state.tutorial.active,
  );
  const [position, setPosition] = useState<TPosition>({
    x: viewportGap,
    y: viewportGap,
    animate: false,
    visible: false,
  });
  const renderedLocally =
    anchor === "notifications-preview" || movement === "kick";
  const enabled =
    shouldShowMascotGuide(pathname) && !tutorialActive && !renderedLocally;

  useLayoutEffect(() => {
    const previousPath = previousPathRef.current;
    previousPathRef.current = pathname;

    if (!shouldShowMascotGuide(previousPath) && enabled) {
      reset();
    }
  }, [enabled, pathname, reset]);

  const updatePosition = useCallback(() => {
    const guide = guideRef.current;
    if (!enabled || !guide) {
      return;
    }

    const width = window.innerWidth;
    const height = window.innerHeight;
    const guideRect = guide.getBoundingClientRect();
    const guideWidth = Math.ceil(guideRect.width);
    const guideHeight = Math.ceil(guideRect.height);

    if (
      guideWidth > width - viewportGap * 2 ||
      guideHeight > height - viewportGap * 2
    ) {
      setPosition((current) => ({ ...current, visible: false }));
      return;
    }

    const target = anchor
      ? document.querySelector<HTMLElement>(
          `[data-mascot-anchor="${anchor}"]`,
        )
      : null;
    const preferred = getPreferredPoint(
      target,
      guideWidth,
      guideHeight,
      width,
      height,
    );
    const protectedRects = collectProtectedRects(guide, width, height);
    const candidates = createCandidates(
      preferred,
      target,
      guideWidth,
      guideHeight,
      width,
      height,
    ).filter((candidate) =>
      isSafePoint(
        candidate,
        guideWidth,
        guideHeight,
        protectedRects,
      ),
    );

    if (candidates.length === 0) {
      setPosition((current) => ({ ...current, visible: false }));
      return;
    }

    const candidateIndex =
      movement === "roam" || movement === "wander" || movement === "scan"
        ? roamStep % Math.min(candidates.length, 5)
        : 0;
    const next = candidates[candidateIndex];

    setPosition((current) => {
      const nextPoint = {
        x: Math.round(next.x),
        y: Math.round(next.y),
      };
      const animate =
        current.visible &&
        isSafePath(
          current,
          nextPoint,
          guideWidth,
          guideHeight,
          protectedRects,
        );

      return {
        ...nextPoint,
        animate,
        visible: true,
      };
    });
  }, [anchor, enabled, movement, roamStep]);

  useLayoutEffect(() => {
    updatePosition();
  }, [message, revision, updatePosition]);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    const interval =
      movement === "scan"
          ? 2400
          : movement === "wander"
            ? 3400
            : movement === "roam"
              ? 7200
              : null;

    if (!interval) {
      return;
    }

    const timerId = window.setInterval(() => {
      setRoamStep((step) => step + 1);
    }, interval);

    return () => window.clearInterval(timerId);
  }, [enabled, movement, revision]);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    let animationFrame = window.requestAnimationFrame(updatePosition);
    const scheduleUpdate = () => {
      window.cancelAnimationFrame(animationFrame);
      animationFrame = window.requestAnimationFrame(updatePosition);
    };
    const observer = new MutationObserver(scheduleUpdate);
    const resizeObserver = new ResizeObserver(scheduleUpdate);

    observer.observe(document.body, {
      characterData: true,
      childList: true,
      subtree: true,
    });
    resizeObserver.observe(document.body);
    if (guideRef.current) {
      resizeObserver.observe(guideRef.current);
    }
    window.addEventListener("resize", scheduleUpdate);
    window.addEventListener("scroll", scheduleUpdate, true);

    return () => {
      window.cancelAnimationFrame(animationFrame);
      observer.disconnect();
      resizeObserver.disconnect();
      window.removeEventListener("resize", scheduleUpdate);
      window.removeEventListener("scroll", scheduleUpdate, true);
    };
  }, [enabled, message, revision, updatePosition]);

  if (!enabled) {
    return null;
  }

  const guideStyle = {
    "--mascot-guide-left": `${position.x}px`,
    "--mascot-guide-top": `${position.y}px`,
  } as CSSProperties;

  return (
    <aside
      aria-live="polite"
      className={clsx(
        styles.guide,
        position.visible ? styles.guide_visible : styles.guide_hidden,
        !position.animate && styles.guide_withoutTransition,
      )}
      ref={guideRef}
      style={guideStyle}
    >
      <Mascot placement="floating" size="small" />
    </aside>
  );
};

export const MascotGuide = memo(MascotGuideComponent);
