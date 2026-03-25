export type LoginRecord = {
  city?: string;
  region?: string;
  country?: string;
  ip_address?: string;
  login_method?: string;
  device_type?: string;
  logged_in_at: string;
};

export type LoginHistoryResponse = {
  records?: LoginRecord[];
};

export type EntrySummary = {
  id: number;
  title: string;
  is_public?: boolean;
};

export type EntryDetailResponse = {
  entry?: EntrySummary;
  content?: string;
  can_edit?: boolean;
};

export type EntryListResponse = {
  entries?: EntrySummary[];
  has_more?: boolean;
  next_offset?: number;
};

export type TagPayload = {
  name: string;
  slug: string;
  description: string;
  sort_order: number;
};

export type Tag = TagPayload & {
  id: number;
  created_at: string;
  updated_at: string;
};

export type TagListResponse = {
  tags?: Tag[];
  has_more?: boolean;
  next_offset?: number;
};

export type SiteSettings = {
  name: string;
  description: string;
  icon_url?: string;
  apple_push_dev_cert?: ApplePushCertificate;
  apple_push_prod_cert?: ApplePushCertificate;
  updated_at?: string;
};

export type ApplePushCertificate = {
  environment: "dev" | "prod";
  file_name: string;
  file_url: string;
  uploaded_at?: string;
};

export type SiteSettingsResponse = ErrorResponse & {
  site?: SiteSettings;
};

export type ErrorResponse = {
  error?: string;
  message?: string;
};

export type IconUploadResponse = ErrorResponse & {
  icon_url?: string;
  site?: SiteSettings;
};

export type PasskeyBeginResponse = ErrorResponse & {
  session_id?: string;
  publicKey: {
    challenge: string | Uint8Array;
    user: {
      id: string | Uint8Array;
    };
    excludeCredentials?: Array<{
      id: string | Uint8Array;
      type: string;
    }>;
  };
};

export type LLMConfigPayload = {
  name: string;
  base_url: string;
  model: string;
  api_key?: string;
  system_prompt: string;
  shared?: boolean;
};

export type LLMConfig = {
  id: number;
  owner_user_id: string;
  share_id: string;
  shared: boolean;
  name: string;
  base_url: string;
  model: string;
  system_prompt: string;
  has_api_key: boolean;
  created_at: string;
  updated_at: string;
};

export type LLMConfigListResponse = ErrorResponse & {
  configs?: LLMConfig[];
  config?: LLMConfig;
};

export type BotPayload = {
  name: string;
  description: string;
  system_prompt: string;
  llm_config_id: number;
};

export type BotUser = {
  id: number;
  owner_user_id: string;
  bot_user_id: string;
  name: string;
  description: string;
  system_prompt: string;
  llm_config_id: number;
  config_name: string;
  created_at: string;
  updated_at: string;
};

export type BotListResponse = ErrorResponse & {
  bots?: BotUser[];
  bot?: BotUser;
};
