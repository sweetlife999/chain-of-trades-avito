import { memo, useEffect } from "react";
import clsx from "clsx";
import { useDispatch } from "react-redux";

import styles from "./Styles.module.scss";
import { useMascot } from "../../../Hooks/useMascot";
import { mascotSettled } from "../../../Features/Mascot/mascotSlice";
import type {
  MascotMode,
  MascotMood,
  MascotMovement,
} from "../../../Features/Mascot/mascot.types";
import type { AuthDispatch } from "../../../Store/store";

type MascotSize = "tiny" | "small" | "medium" | "large";
type MascotPlacement = "landing" | "chat" | "inline" | "floating";

type TProps = {
  bubbleAction?: {
    label: string;
    meta?: string;
    onClick: () => void;
  };
  size?: MascotSize;
  placement?: MascotPlacement;
  className?: string;
  showBubble?: boolean;
  message?: string | null;
  label?: string;
  mode?: MascotMode;
  mood?: MascotMood;
  movement?: MascotMovement;
};

const defaultMouth = "M105 111 Q120 121 135 111";
const mouthByMood: Partial<Record<MascotMood, string>> = {
  happy: "M101 108 Q120 127 139 108",
  celebrate: "M99 106 Q120 130 141 106",
  concerned: "M105 119 Q120 108 135 119",
  excited: "M99 106 Q120 130 141 106",
  sad: "M104 121 Q120 108 136 121",
  bored: "M108 116 Q120 114 132 116",
  reading: "M108 114 Q120 118 132 114",
  angry: "M105 118 Q120 111 135 118",
};

const MascotComponent = ({
  bubbleAction,
  size = "medium",
  placement = "inline",
  className,
  showBubble = true,
  message,
  label = "Уми — помощник по обмену",
  mode: modeOverride,
  mood: moodOverride,
  movement: movementOverride,
}: TProps) => {
  const dispatch = useDispatch<AuthDispatch>();
  const {
    mood,
    mode,
    movement,
    message: stateMessage,
    durationMs,
    revision,
  } = useMascot();
  const visibleMode = modeOverride ?? mode;
  const visibleMood = moodOverride ?? mood;
  const visibleMovement = movementOverride ?? movement;
  const eyeY = visibleMood === "angry" ? 76 : 83;
  const visibleMessage = message === undefined ? stateMessage : message;
  const bubbleVisible =
    showBubble &&
    Boolean(visibleMessage) &&
    (message !== undefined || visibleMode !== "ambient");

  useEffect(() => {
    if (!durationMs) {
      return;
    }

    const timeout = window.setTimeout(
      () => dispatch(mascotSettled(revision)),
      durationMs,
    );

    return () => window.clearTimeout(timeout);
  }, [dispatch, durationMs, revision]);

  return (
    <div
      className={clsx(
        styles.mascot,
        styles[`mascot_size_${size}`],
        styles[`mascot_placement_${placement}`],
        styles[`mascot_mood_${visibleMood}`],
        styles[`mascot_movement_${visibleMovement}`],
        className,
      )}
      data-mascot-mood={visibleMood}
      data-mascot-movement={visibleMovement}
    >
      {bubbleVisible && (
        <div
          className={clsx(
            styles.mascot__bubble,
            styles[`mascot__bubble_${visibleMode}`],
          )}
          data-mascot-bubble
        >
          <strong className={styles.mascot__bubbleName}>Уми</strong>
          <span className={styles.mascot__bubbleText}>{visibleMessage}</span>
          {bubbleAction && (
            <span className={styles.mascot__bubbleFooter}>
              {bubbleAction.meta && (
                <small className={styles.mascot__bubbleMeta}>
                  {bubbleAction.meta}
                </small>
              )}
              <button
                className={styles.mascot__bubbleAction}
                type="button"
                onClick={bubbleAction.onClick}
              >
                {bubbleAction.label}
              </button>
            </span>
          )}
        </div>
      )}

      <div
        aria-label={label}
        className={styles.mascot__stage}
        data-mascot-stage
        role="img"
      >
        <span className={styles.mascot__shadow} aria-hidden="true" />
        <span className={styles.mascot__confetti} aria-hidden="true">
          <i /><i /><i /><i /><i /><i />
        </span>

        <svg
          className={styles.mascot__robot}
          viewBox="0 0 240 280"
          aria-hidden="true"
        >
          <g className={styles.mascot__antenna}>
            <path d="M120 47V31" />
            <circle cx="120" cy="23" r="9" />
            <path className={styles.mascot__antennaGlow} d="M120 47V31" />
          </g>

          <g className={styles.mascot__bodyGroup}>
            <g className={styles.mascot__leftArm}>
              <rect x="28" y="126" width="34" height="86" rx="17" />
              <circle cx="45" cy="211" r="19" />
            </g>
            <g className={styles.mascot__rightArm}>
              <rect x="178" y="126" width="34" height="86" rx="17" />
              <circle cx="195" cy="211" r="19" />
            </g>

            <rect className={styles.mascot__torso} x="62" y="118" width="116" height="116" rx="34" />
            <rect className={styles.mascot__chest} x="85" y="150" width="70" height="60" rx="18" />
            <text className={styles.mascot__letter} x="120" y="189" textAnchor="middle">Уми</text>

            <g className={styles.mascot__leftLeg}>
              <rect x="79" y="218" width="30" height="43" rx="15" />
              <rect x="67" y="250" width="48" height="20" rx="10" />
            </g>
            <g className={styles.mascot__rightLeg}>
              <rect x="131" y="218" width="30" height="43" rx="15" />
              <rect x="125" y="250" width="48" height="20" rx="10" />
            </g>
          </g>

          <g className={styles.mascot__head}>
            <rect className={styles.mascot__headShell} x="49" y="48" width="142" height="96" rx="38" />
            <rect className={styles.mascot__face} x="63" y="63" width="114" height="65" rx="26" />
            <g className={styles.mascot__eyes}>
              <rect className={styles.mascot__eye} x="84" y={eyeY} width="16" height="20" rx="8" />
              <rect className={styles.mascot__eye} x="140" y={eyeY} width="16" height="20" rx="8" />
            </g>
            <path
              className={styles.mascot__mouth}
              d={mouthByMood[visibleMood] ?? defaultMouth}
            />
            <circle className={styles.mascot__cheek} cx="82" cy="111" r="4" />
            <circle className={styles.mascot__cheek} cx="158" cy="111" r="4" />
          </g>

          <g className={styles.mascot__thinkingDots}>
            <circle cx="194" cy="58" r="5" />
            <circle cx="211" cy="43" r="7" />
            <circle cx="229" cy="24" r="9" />
          </g>
        </svg>
      </div>
    </div>
  );
};

export const Mascot = memo(MascotComponent);
