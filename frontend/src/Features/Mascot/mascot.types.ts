export type MascotMood =
  | "idle"
  | "hello"
  | "listening"
  | "thinking"
  | "hint"
  | "happy"
  | "celebrate"
  | "concerned";

export type MascotMode = "ambient" | "attention" | "dialog";

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
  | "ERROR";

export type MascotReaction = {
  mood: MascotMood;
  mode: MascotMode;
  message: string | null;
  durationMs: number | null;
};
