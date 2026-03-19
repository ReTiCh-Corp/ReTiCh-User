-- Supprimer le trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Supprimer les nouveaux champs de profil
ALTER TABLE profiles DROP COLUMN IF EXISTS phone;
ALTER TABLE profiles DROP COLUMN IF EXISTS gender;
ALTER TABLE profiles DROP COLUMN IF EXISTS last_name;
ALTER TABLE profiles DROP COLUMN IF EXISTS first_name;

-- Remettre username NOT NULL
ALTER TABLE profiles ALTER COLUMN username SET NOT NULL;

-- Recréer la colonne email dans profiles
ALTER TABLE profiles ADD COLUMN email VARCHAR(255) UNIQUE;

-- Migrer les emails de users vers profiles
UPDATE profiles
SET email = u.email
FROM users u
WHERE profiles.id = u.id;

-- Supprimer la FK
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS fk_profiles_user;

-- Supprimer la table users
DROP TABLE IF EXISTS users;
