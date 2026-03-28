export type UserProfile = {
  user_id: string;
  username: string;
  role?: string;
  email?: string;
  email_verified?: boolean;
  icon_url?: string;
  bio?: string;
  is_online?: boolean;
  device_type?: string;
  last_seen_at?: string;
};

export type EmailVerificationResponse = {
  message?: string;
  error?: string;
  email?: string;
  email_verified?: boolean;
};
