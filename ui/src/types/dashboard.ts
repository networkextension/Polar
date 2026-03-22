export type LoginRecord = {
  city?: string;
  region?: string;
  country?: string;
  ip_address?: string;
  login_method?: string;
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
  updated_at?: string;
};

export type SiteSettingsResponse = ErrorResponse & {
  site?: SiteSettings;
};

export type ErrorResponse = {
  error?: string;
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
