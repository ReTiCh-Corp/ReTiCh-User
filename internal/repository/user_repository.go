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

// Create insère un nouvel utilisateur dans la table users.
func (r *UserRepository) Create(id, email string) (*model.User, error) {
	user := &model.User{}
	query := `
		INSERT INTO users (id, email)
		VALUES ($1, $2)
		RETURNING id, email, onboarding_completed, created_at, updated_at`

	err := r.db.QueryRow(query, id, email).Scan(
		&user.ID,
		&user.Email,
		&user.OnboardingCompleted,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByEmail retourne un utilisateur par son email.
func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	user := &model.User{}
	query := `
		SELECT id, email, onboarding_completed, created_at, updated_at
		FROM users
		WHERE email = $1`

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.OnboardingCompleted,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateByID(id string, req *model.UpdateProfileRequest) (*model.Profile, error) {
	profile := &model.Profile{}
	query := `
		UPDATE profiles
		SET username = $2, display_name = $3, avatar_url = $4, bio = $5,
		    status = $6, custom_status = $7, first_name = $8, last_name = $9,
		    gender = $10, phone = $11
		WHERE id = $1
		RETURNING id, username, display_name, avatar_url, bio, status, custom_status,
		          first_name, last_name, gender, phone, last_seen_at, created_at, updated_at`

	err := r.db.QueryRow(query, id, req.Username, req.DisplayName, req.AvatarURL,
		req.Bio, req.Status, req.CustomStatus, req.FirstName, req.LastName,
		req.Gender, req.Phone).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Status,
		&profile.CustomStatus,
		&profile.FirstName,
		&profile.LastName,
		&profile.Gender,
		&profile.Phone,
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
func (r *UserRepository) UpdateAvatarURL(id, avatarURL string) (*model.Profile, error) {
	profile := &model.Profile{}
	query := `
		UPDATE profiles
		SET avatar_url = $2
		WHERE id = $1
		RETURNING id, username, display_name, avatar_url, bio, status, custom_status,
		          first_name, last_name, gender, phone, last_seen_at, created_at, updated_at`

	err := r.db.QueryRow(query, id, avatarURL).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Status,
		&profile.CustomStatus,
		&profile.FirstName,
		&profile.LastName,
		&profile.Gender,
		&profile.Phone,
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

// listQueries contient les requêtes SELECT pré-construites pour chaque option de tri.
// Les requêtes font un LEFT JOIN entre users et profiles pour récupérer l'email.
var listQueries = map[string]string{
	"username":    `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id ORDER BY p.username ASC LIMIT $1 OFFSET $2`,
	"-username":   `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id ORDER BY p.username DESC LIMIT $1 OFFSET $2`,
	"created_at":  `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id ORDER BY u.created_at ASC LIMIT $1 OFFSET $2`,
	"-created_at": `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id ORDER BY u.created_at DESC LIMIT $1 OFFSET $2`,
}

var searchQueries = map[string]string{
	"username":    `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id WHERE p.username ILIKE $1 OR u.email ILIKE $1 ORDER BY p.username ASC LIMIT $2 OFFSET $3`,
	"-username":   `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id WHERE p.username ILIKE $1 OR u.email ILIKE $1 ORDER BY p.username DESC LIMIT $2 OFFSET $3`,
	"created_at":  `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id WHERE p.username ILIKE $1 OR u.email ILIKE $1 ORDER BY u.created_at ASC LIMIT $2 OFFSET $3`,
	"-created_at": `SELECT u.id, u.email, COALESCE(p.username, ''), u.onboarding_completed FROM users u LEFT JOIN profiles p ON p.id = u.id WHERE p.username ILIKE $1 OR u.email ILIKE $1 ORDER BY u.created_at DESC LIMIT $2 OFFSET $3`,
}

// List retourne une page de UserSummary et le total correspondant.
func (r *UserRepository) List(search, sort string, limit, offset int) ([]*model.UserSummary, int, error) {
	var total int
	var rows *sql.Rows
	var err error

	if search != "" {
		pattern := "%" + search + "%"
		err = r.db.QueryRow(
			`SELECT COUNT(*) FROM users u LEFT JOIN profiles p ON p.id = u.id WHERE p.username ILIKE $1 OR u.email ILIKE $1`,
			pattern,
		).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		q, ok := searchQueries[sort]
		if !ok {
			q = searchQueries["username"]
		}
		rows, err = r.db.Query(q, pattern, limit, offset)
	} else {
		err = r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		q, ok := listQueries[sort]
		if !ok {
			q = listQueries["username"]
		}
		rows, err = r.db.Query(q, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.UserSummary
	for rows.Next() {
		u := &model.UserSummary{}
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.OnboardingCompleted); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetByUsername retourne un profil par son username, ou ErrNotFound.
func (r *UserRepository) GetByUsername(username string) (*model.Profile, error) {
	profile := &model.Profile{}
	query := `
		SELECT id, username, display_name, avatar_url, bio, status, custom_status,
		       first_name, last_name, gender, phone, last_seen_at, created_at, updated_at
		FROM profiles
		WHERE username = $1`

	err := r.db.QueryRow(query, username).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.AvatarURL,
		&profile.Bio,
		&profile.Status,
		&profile.CustomStatus,
		&profile.FirstName,
		&profile.LastName,
		&profile.Gender,
		&profile.Phone,
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
		SELECT id, username, display_name, avatar_url, bio, status, custom_status,
		       first_name, last_name, gender, phone, last_seen_at, created_at, updated_at
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
		&profile.FirstName,
		&profile.LastName,
		&profile.Gender,
		&profile.Phone,
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
