import { useState } from "react";
import { Flag } from "lucide-react";

import styles from "./Styles.module.scss";
import { ReportMessagePopup } from "../ReportMessagePopup/ReportMessagePopup";

type TReportMessageButtonProps = {
  className?: string;
  messageId: string;
};

export const ReportMessageButton = ({
  className = "",
  messageId,
}: TReportMessageButtonProps) => {
  const [popupOpen, setPopupOpen] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  return (
    <>
      <button
        aria-label={submitted ? "Жалоба отправлена" : "Пожаловаться на сообщение"}
        className={`${styles.reportButton} ${className}`}
        disabled={submitted}
        title={submitted ? "Жалоба отправлена" : "Пожаловаться"}
        type="button"
        onClick={() => setPopupOpen(true)}
      >
        <Flag aria-hidden="true" size={15} />
      </button>

      {popupOpen && (
        <ReportMessagePopup
          messageId={messageId}
          onClose={() => setPopupOpen(false)}
          onSubmitted={() => setSubmitted(true)}
        />
      )}
    </>
  );
};
