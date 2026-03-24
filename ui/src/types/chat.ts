export type ChatSummary = {
  id: string;
  other_user_id: string;
  other_username: string;
  other_user_icon?: string;
  other_user_online?: boolean;
  other_user_device_type?: string;
  other_user_last_seen_at?: string;
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
  message_type?: string;
  content: string;
  markdown_entry_id?: number;
  markdown_title?: string;
  deleted?: boolean;
  created_at: string;
};

export type ChatMessagesResponse = {
  messages?: ChatMessage[];
};

export type SharedMarkdownResponse = {
  entry?: {
    id: number;
    title?: string;
    is_public?: boolean;
  };
  content?: string;
  can_edit?: boolean;
  error?: string;
};

export type ChatEventPayload = {
  type?: string;
  chat_id?: string;
  user_id?: string;
  online?: boolean;
  device_type?: string;
  last_seen_at?: string;
};
