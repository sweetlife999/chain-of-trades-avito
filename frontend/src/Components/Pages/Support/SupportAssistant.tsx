import { memo } from "react";

import styles from "./Styles.module.scss";
import type {
  MascotMood,
  MascotMovement,
} from "../../../Features/Mascot/mascot.types";
import { Mascot } from "../../UI/Mascot/Mascot";
import { supportHints } from "./supportHints";

type TSupportAssistantProps = {
  level?: number;
  mood: MascotMood;
  movement: MascotMovement;
  onHintPreview: () => void;
  onHintSelect: (hint: string) => void;
};

const SupportAssistantComponent = ({
  level,
  mood,
  movement,
  onHintPreview,
  onHintSelect,
}: TSupportAssistantProps) => (
  <section
    aria-label="Быстрые подсказки Уми"
    className={styles.support__assistant}
  >
    <Mascot
      className={styles.support__assistantMascot}
      level={level}
      message={null}
      mood={mood}
      movement={movement}
      placement="chat"
      showBubble={false}
      size="small"
    />
    <div className={styles.support__hints}>
      {supportHints.map((hint) => (
        <button
          className={styles.support__hint}
          key={hint}
          type="button"
          onClick={() => onHintSelect(hint)}
          onFocus={onHintPreview}
          onMouseEnter={onHintPreview}
        >
          {hint}
        </button>
      ))}
    </div>
  </section>
);

export const SupportAssistant = memo(SupportAssistantComponent);
