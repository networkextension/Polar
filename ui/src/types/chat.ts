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
  is_implicit_friend?: boolean;
  reply_required?: boolean;
  reply_required_message?: string;
};

export type ChatListResponse = {
  chats?: ChatSummary[];
};

export type StartChatResponse = {
  chat?: ChatSummary;
  error?: string;
};

export type ChatMessageAttachment = {
  url: string;
  file_name: string;
  size: number;
  mime_type: string;
  thumbnail_url?: string;
  width?: number;
  height?: number;
};

export type ChatMessage = {
  id: string;
  llm_thread_id?: number;
  sender_id: string;
  sender_username: string;
  sender_icon?: string;
  message_type?: string;
  failed?: boolean;
  content: string;
  markdown_entry_id?: number;
  markdown_title?: string;
  attachment?: ChatMessageAttachment;
  deleted?: boolean;
  created_at: string;
};

export type ChatMessagesResponse = {
  messages?: ChatMessage[];
  active_thread?: LLMThread;
  active_thread_id?: number;
  blocked?: boolean;
  block_message?: string;
  is_implicit_friend?: boolean;
  reply_required?: boolean;
  reply_required_message?: string;
};

export type LLMThread = {
  id: number;
  chat_thread_id: number;
  owner_user_id: string;
  bot_user_id: string;
  llm_config_id?: number;
  config_name?: string;
  config_model?: string;
  title: string;
  created_at: string;
  updated_at: string;
  last_message_at?: string;
};

export type ChatLLMConfig = {
  id: number;
  owner_user_id: string;
  name: string;
  model: string;
  shared: boolean;
};

export type LLMThreadListResponse = {
  threads?: LLMThread[];
  thread?: LLMThread;
  active_thread?: LLMThread;
  message?: string;
  error?: string;
};

export type ChatLLMConfigListResponse = {
  configs?: ChatLLMConfig[];
  error?: string;
};

export type SendMessageResponse = {
  message?: string;
  id?: string;
  error?: string;
  code?: string;
  active_thread?: LLMThread;
  is_implicit_friend?: boolean;
  reply_required?: boolean;
  reply_required_message?: string;
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
  message?: ChatMessage;
  message_id?: string;
  user_id?: string;
  read_at?: string;
  deleted_at?: string;
  online?: boolean;
  device_type?: string;
  last_seen_at?: string;
};
