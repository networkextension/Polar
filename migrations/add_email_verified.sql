-- Add email_verified column to users table
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;
