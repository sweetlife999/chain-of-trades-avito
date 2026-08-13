export type MascotMood =
  | "idle"
  | "hello"
  | "listening"
  | "thinking"
  | "hint"
  | "pointing"
  | "happy"
  | "excited"
  | "celebrate"
  | "concerned"
  | "sad"
  | "bored"
  | "reading"
  | "angry";

export type MascotMode = "ambient" | "attention" | "dialog";

export type MascotMovement =
  | "roam"
  | "approach"
  | "wander"
  | "run"
  | "scan"
  | "kick";

export type MascotAnchor =
  | "form-submit"
  | "items-empty"
  | "exchanges-empty"
  | "chains-list"
  | "notifications-trigger"
  | "notifications-preview"
  | "notifications-panel"
  | "notifications-list"
  | "latest-review"
  | "blocked-users";

export type MascotEvent =
  | "CHAT_OPENED"
  | "USER_TYPING"
  | "HINT_SHOWN"
  | "HINT_SELECTED"
  | "MESSAGE_SENT"
  | "MESSAGE_SENT_SUCCESS"
  | "MESSAGE_RECEIVED"
  | "PARTICIPANT_ACCEPTED"
  | "PARTICIPANT_DECLINED"
  | "PARTICIPANT_DELIVERED"
  | "EXCHANGE_CONFIRMED"
  | "EXCHANGE_DELIVERING"
  | "EXCHANGE_DELIVERED"
  | "EXCHANGE_COMPLETED"
  | "AUTH_FORM_READY"
  | "FORM_SUBMIT"
  | "FORM_SUCCESS"
  | "FORM_ERROR"
  | "EMPTY_ITEMS"
  | "EMPTY_EXCHANGES"
  | "EMPTY_CHAINS"
  | "CHAIN_AVAILABLE"
  | "NEW_NOTIFICATION"
  | "NOTIFICATIONS_PREVIEW_OPENED"
  | "NOTIFICATIONS_OPENED"
  | "NEW_REVIEW"
  | "BLOCKED_USER_HOVERED"
  | "ERROR";

export type MascotReaction = {
  mood: MascotMood;
  mode: MascotMode;
  message: string | null;
  durationMs: number | null;
  anchor: MascotAnchor | null;
  movement: MascotMovement;
};
