import {
  memo,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import { useSelector } from "react-redux";
import { useLocation, useNavigate } from "react-router-dom";

import styles from "./Styles.module.scss";
import {
  completeTutorial,
  readTutorialProgress,
  saveTutorialProgress,
  tutorialSteps,
} from "../../../Features/Tutorial/tutorial";
import {
  tutorialAdvanced,
  tutorialFinished,
  tutorialStarted,
  tutorialStopped,
} from "../../../Features/Tutorial/tutorialSlice";
import { useAuthDispatch, useAuthSelector } from "../../../Hooks/useAuthDispatch";
import { useMascot } from "../../../Hooks/useMascot";
import type { RootState } from "../../../Store/store";
import { Mascot } from "../Mascot/Mascot";

const viewportGap = 12;
const targetGap = 8;
const guideGap = 18;

type TRect = {
  bottom: number;
  height: number;
  left: number;
  right: number;
  top: number;
  width: number;
};

type TPoint = {
  x: number;
  y: number;
};

type TView = {
  guide: TPoint;
  target: TRect | null;
};

const clamp = (value: number, min: number, max: number) =>
  Math.min(Math.max(value, min), Math.max(min, max));

const isVisible = (element: HTMLElement) => {
  const rect = element.getBoundingClientRect();
  const style = window.getComputedStyle(element);
  return (
    rect.width > 0 &&
    rect.height > 0 &&
    style.display !== "none" &&
    style.visibility !== "hidden" &&
    Number(style.opacity) !== 0
  );
};

const findTarget = (target: string) =>
  Array.from(
    document.querySelectorAll<HTMLElement>(
      `[data-tutorial-target="${target}"]`,
    ),
  ).find(isVisible) ?? null;

const getGuidePosition = (
  target: TRect,
  guideWidth: number,
  guideHeight: number,
) => {
  const width = window.innerWidth;
  const height = window.innerHeight;
  const centeredX = target.left + target.width / 2 - guideWidth / 2;
  const centeredY = target.top + target.height / 2 - guideHeight / 2;
  const candidates: TPoint[] = [
    { x: centeredX, y: target.bottom + guideGap },
    { x: target.right + guideGap, y: centeredY },
    { x: target.left - guideWidth - guideGap, y: centeredY },
    { x: centeredX, y: target.top - guideHeight - guideGap },
  ];
  const fitting = candidates.find(
    ({ x, y }) =>
      x >= viewportGap &&
      y >= viewportGap &&
      x + guideWidth <= width - viewportGap &&
      y + guideHeight <= height - viewportGap,
  );
  const selected = fitting ?? candidates[0];

  return {
    x: Math.round(
      clamp(selected.x, viewportGap, width - guideWidth - viewportGap),
    ),
    y: Math.round(
      clamp(selected.y, viewportGap, height - guideHeight - viewportGap),
    ),
  };
};

const sameRect = (first: TRect | null, second: TRect | null) => {
  if (!first || !second) {
    return first === second;
  }

  return (
    first.left === second.left &&
    first.top === second.top &&
    first.width === second.width &&
    first.height === second.height
  );
};

const OnboardingTutorialComponent = () => {
  const dispatch = useAuthDispatch();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const { isAuth, user } = useAuthSelector();
  const tutorial = useSelector((state: RootState) => state.tutorial);
  const { reset } = useMascot();
  const guideRef = useRef<HTMLElement>(null);
  const [view, setView] = useState<TView>({
    guide: { x: viewportGap, y: viewportGap },
    target: null,
  });
  const step = tutorialSteps[tutorial.step];

  useEffect(() => {
    if (!isAuth || !user) {
      if (tutorial.active) {
        dispatch(tutorialStopped());
      }
      return;
    }

    if (tutorial.active) {
      if (tutorial.userId !== user.id) {
        dispatch(tutorialStopped());
      }
      return;
    }

    const savedStep = readTutorialProgress(user.id);
    if (savedStep !== null) {
      dispatch(tutorialStarted({ step: savedStep, userId: user.id }));
    }
  }, [dispatch, isAuth, tutorial.active, tutorial.userId, user]);

  useEffect(() => {
    if (tutorial.active && pathname !== "/feed") {
      navigate("/feed", { replace: true });
    }
  }, [navigate, pathname, tutorial.active]);

  const handleAdvance = useCallback(() => {
    if (!tutorial.active || !tutorial.userId) {
      return;
    }

    const nextStep = tutorial.step + 1;
    if (nextStep >= tutorialSteps.length) {
      completeTutorial(tutorial.userId);
      dispatch(tutorialFinished());
      reset();
      return;
    }

    saveTutorialProgress(tutorial.userId, nextStep);
    dispatch(tutorialAdvanced());
  }, [dispatch, reset, tutorial.active, tutorial.step, tutorial.userId]);

  useLayoutEffect(() => {
    if (!tutorial.active || !step) {
      return;
    }

    let animationFrame = 0;
    const resizeObserver = new ResizeObserver(() => scheduleSync());

    const sync = () => {
      const target = findTarget(step.target);
      const guide = guideRef.current;
      if (!target || !guide) {
        setView((current) =>
          current.target === null ? current : { ...current, target: null },
        );
        return;
      }

      const rawTargetRect = target.getBoundingClientRect();
      const targetRect: TRect = {
        top: Math.round(rawTargetRect.top - targetGap),
        right: Math.round(rawTargetRect.right + targetGap),
        bottom: Math.round(rawTargetRect.bottom + targetGap),
        left: Math.round(rawTargetRect.left - targetGap),
        width: Math.round(rawTargetRect.width + targetGap * 2),
        height: Math.round(rawTargetRect.height + targetGap * 2),
      };
      const guideRect = guide.getBoundingClientRect();
      const guidePosition = getGuidePosition(
        targetRect,
        Math.ceil(guideRect.width),
        Math.ceil(guideRect.height),
      );

      resizeObserver.disconnect();
      resizeObserver.observe(target);
      resizeObserver.observe(guide);
      setView((current) =>
        sameRect(current.target, targetRect) &&
        current.guide.x === guidePosition.x &&
        current.guide.y === guidePosition.y
          ? current
          : { guide: guidePosition, target: targetRect },
      );
    };

    function scheduleSync() {
      window.cancelAnimationFrame(animationFrame);
      animationFrame = window.requestAnimationFrame(sync);
    }

    const mutationObserver = new MutationObserver(scheduleSync);
    mutationObserver.observe(document.body, {
      attributeFilter: ["class", "style"],
      attributes: true,
      childList: true,
      subtree: true,
    });
    window.addEventListener("resize", scheduleSync);
    window.addEventListener("scroll", scheduleSync, true);
    const firstRetry = window.setTimeout(scheduleSync, 80);
    const secondRetry = window.setTimeout(scheduleSync, 240);
    scheduleSync();

    return () => {
      window.cancelAnimationFrame(animationFrame);
      window.clearTimeout(firstRetry);
      window.clearTimeout(secondRetry);
      mutationObserver.disconnect();
      resizeObserver.disconnect();
      window.removeEventListener("resize", scheduleSync);
      window.removeEventListener("scroll", scheduleSync, true);
    };
  }, [step, tutorial.active]);

  if (!tutorial.active || !step || !user) {
    return null;
  }

  const guideStyle = {
    "--tutorial-guide-left": `${view.guide.x}px`,
    "--tutorial-guide-top": `${view.guide.y}px`,
  } as CSSProperties;
  const spotlightStyle = view.target
    ? {
        left: `${view.target.left}px`,
        top: `${view.target.top}px`,
        width: `${view.target.width}px`,
        height: `${view.target.height}px`,
      }
    : undefined;

  return (
    <section
      aria-label="Знакомство с интерфейсом"
      aria-modal="true"
      className={styles.tutorial}
      role="dialog"
    >
      <button
        aria-label="Перейти к следующему шагу знакомства"
        className={styles.tutorial__backdrop}
        type="button"
        onClick={handleAdvance}
      />
      {view.target && (
        <span
          aria-hidden="true"
          className={styles.tutorial__spotlight}
          style={spotlightStyle}
        />
      )}
      <aside
        aria-live="polite"
        className={`${styles.tutorial__guide} ${
          view.target ? styles.tutorial__guide_visible : ""
        }`}
        ref={guideRef}
        style={guideStyle}
      >
        <Mascot
          bubbleAction={{
            label:
              tutorial.step === tutorialSteps.length - 1 ? "Готово" : "Далее",
            meta: `${tutorial.step + 1} из ${tutorialSteps.length}`,
            onClick: handleAdvance,
          }}
          className={styles.tutorial__mascot}
          message={step.message}
          mode="attention"
          mood="pointing"
          movement="approach"
          placement="floating"
          size="small"
        />
      </aside>
    </section>
  );
};

export const OnboardingTutorial = memo(OnboardingTutorialComponent);
