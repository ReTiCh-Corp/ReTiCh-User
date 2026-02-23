-- Rollback User Service Schema

DROP TRIGGER IF EXISTS update_user_settings_updated_at ON user_settings;
DROP TRIGGER IF EXISTS update_profiles_updated_at ON profiles;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS blocked_users;
DROP TABLE IF EXISTS friend_requests;
DROP TABLE IF EXISTS contacts;
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS profiles;

DROP TYPE IF EXISTS friend_request_status;
DROP TYPE IF EXISTS user_status;
