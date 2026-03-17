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

// allowedSortColumns liste les valeurs autorisées pour le paramètre sort.
// La clé est la valeur reçue du client, la valeur est la clause ORDER BY correspondante.
// Le préfixe "-" indique un tri descendant.
var allowedSortColumns = map[string]string{
	"username":    "username ASC",
	"-username":   "username DESC",
	"created_at":  "created_at ASC",
	"-created_at": "created_at DESC",
}

// List retourne une page de UserSummary et le total correspondant.
// Si search est non vide, filtre sur username et email (ILIKE).
// sort doit être une des valeurs de allowedSortColumns ; défaut : "username ASC".
func (r *UserRepository) List(search, sort string, limit, offset int) ([]*model.UserSummary, int, error) {
	orderBy, ok := allowedSortColumns[sort]
	if !ok {
		orderBy = "username ASC"
	}

	var total int
	var rows *sql.Rows
	var err error

	if search != "" {
		pattern := "%" + search + "%"
		err = r.db.QueryRow(
			`SELECT COUNT(*) FROM profiles WHERE username ILIKE $1 OR email ILIKE $1`,
			pattern,
		).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = r.db.Query(
			`SELECT id, COALESCE(email, ''), username FROM profiles
			WHERE username ILIKE $1 OR email ILIKE $1
			ORDER BY `+orderBy+`
			LIMIT $2 OFFSET $3`,
			pattern, limit, offset,
		)
	} else {
		err = r.db.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = r.db.Query(
			`SELECT id, COALESCE(email, ''), username FROM profiles
			ORDER BY `+orderBy+`
			LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.UserSummary
	for rows.Next() {
		u := &model.UserSummary{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Username); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
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
