-- Créer la table users (identité, séparée du profil)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    onboarding_completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Migrer les données existantes de profiles vers users
INSERT INTO users (id, email, onboarding_completed, created_at, updated_at)
SELECT id, email, TRUE, created_at, updated_at
FROM profiles
WHERE email IS NOT NULL;

-- Ajouter la FK : profiles.id → users.id
ALTER TABLE profiles
    ADD CONSTRAINT fk_profiles_user
    FOREIGN KEY (id) REFERENCES users(id) ON DELETE CASCADE;

-- Supprimer email de profiles (maintenant dans users)
ALTER TABLE profiles DROP COLUMN email;

-- username nullable (pas encore connu à la création via OAuth)
ALTER TABLE profiles ALTER COLUMN username DROP NOT NULL;

-- Nouveaux champs de profil pour l'onboarding
ALTER TABLE profiles ADD COLUMN first_name VARCHAR(100);
ALTER TABLE profiles ADD COLUMN last_name VARCHAR(100);
ALTER TABLE profiles ADD COLUMN gender VARCHAR(20);
ALTER TABLE profiles ADD COLUMN phone VARCHAR(20);

-- Trigger updated_at sur la table users
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
