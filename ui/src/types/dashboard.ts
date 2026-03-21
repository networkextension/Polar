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
};

export type EntryDetailResponse = {
  entry?: EntrySummary;
  content?: string;
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

export type ErrorResponse = {
  error?: string;
};

export type IconUploadResponse = ErrorResponse & {
  icon_url?: string;
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
