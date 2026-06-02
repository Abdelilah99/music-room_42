ALTER TABLE users 
DROP COLUMN IF EXISTS public_info,
DROP COLUMN IF EXISTS friends_info,
DROP COLUMN IF EXISTS private_info,
DROP COLUMN IF EXISTS music_preferences;