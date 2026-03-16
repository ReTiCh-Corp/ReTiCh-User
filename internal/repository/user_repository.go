package repository

import (
	"database/sql"
	"errors"

	"github.com/retich-corp/user/internal/model"
)

var ErrNotFound = errors.New("user not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) UpdateByID(id string, req *model.UpdateProfileRequest) (*model.Profile, error) {
	profile := &model.Profile{}
	query := `
		UPDATE profiles
		SET username = $2, display_name = $3, avatar_url = $4, bio = $5, status = $6, custom_status = $7
		WHERE id = $1
		RETURNING id, username, display_name, avatar_url, bio, status, custom_status, last_seen_at, created_at, updated_at`

	err := r.db.QueryRow(query, id, req.Username, req.DisplayName, req.AvatarURL, req.Bio, req.Status, req.CustomStatus).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Status,
		&profile.CustomStatus,
		&profile.LastSeenAt,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// UpdateAvatarURL met à jour uniquement la colonne avatar_url pour l'utilisateur donné.
// Retourne le profil complet après mise à jour grâce au RETURNING, sans SELECT supplémentaire.
func (r *UserRepository) UpdateAvatarURL(id, avatarURL string) (*model.Profile, error) {
	profile := &model.Profile{}
	query := `
		UPDATE profiles
		SET avatar_url = $2
		WHERE id = $1
		RETURNING id, username, display_name, avatar_url, bio, status, custom_status, last_seen_at, created_at, updated_at`

	err := r.db.QueryRow(query, id, avatarURL).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Status,
		&profile.CustomStatus,
		&profile.LastSeenAt,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *UserRepository) GetByID(id string) (*model.Profile, error) {
	profile := &model.Profile{}
	query := `
		SELECT id, username, display_name, avatar_url, bio, status, custom_status, last_seen_at, created_at, updated_at
		FROM profiles
		WHERE id = $1`

	err := r.db.QueryRow(query, id).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Status,
		&profile.CustomStatus,
		&profile.LastSeenAt,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return profile, nil
}
