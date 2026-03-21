export type ChatSummary = {
  id: string;
  other_username: string;
  other_user_icon?: string;
  unread_count?: number;
  last_message?: string;
  last_message_at?: string;
};

export type ChatListResponse = {
  chats?: ChatSummary[];
};

export type StartChatResponse = {
  chat?: ChatSummary;
  error?: string;
};

export type ChatMessage = {
  id: string;
  sender_id: string;
  sender_username: string;
  sender_icon?: string;
  content: string;
  deleted?: boolean;
  created_at: string;
};

export type ChatMessagesResponse = {
  messages?: ChatMessage[];
};

export type ChatEventPayload = {
  type?: string;
  chat_id?: string;
};
