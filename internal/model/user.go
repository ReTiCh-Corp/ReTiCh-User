package model

import "time"

// User représente l'identité d'un utilisateur (table users).
type User struct {
	ID                  string    `json:"id"`
	Email               string    `json:"email"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CreateUserRequest est le body attendu par POST /users.
type CreateUserRequest struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// CreateUserResponse est la réponse retournée par POST /users.
type CreateUserResponse struct {
	ID                  string `json:"id"`
	Email               string `json:"email"`
	OnboardingCompleted bool   `json:"onboarding_completed"`
	IsNewUser           bool   `json:"is_new_user"`
}

// UserSummary est la représentation allégée retournée par le endpoint de listing.
type UserSummary struct {
	ID                  string `json:"id"`
	Email               string `json:"email"`
	Username            string `json:"username"`
	OnboardingCompleted bool   `json:"onboarding_completed"`
}

// Profile représente le profil complet d'un utilisateur (table profiles).
type Profile struct {
	ID           string     `json:"id"`
	Username     *string    `json:"username"`
	DisplayName  *string    `json:"display_name"`
	AvatarURL    *string    `json:"avatar_url"`
	Bio          *string    `json:"bio"`
	Status       string     `json:"status"`
	CustomStatus *string    `json:"custom_status"`
	FirstName    *string    `json:"first_name"`
	LastName     *string    `json:"last_name"`
	Gender       *string    `json:"gender"`
	Phone        *string    `json:"phone"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// UpdateProfileRequest contient les champs modifiables par le client.
// Les champs nullables sont des pointeurs : envoyer null les efface en base.
type UpdateProfileRequest struct {
	Username     string  `json:"username"`
	DisplayName  *string `json:"display_name"`
	AvatarURL    *string `json:"avatar_url"`
	Bio          *string `json:"bio"`
	Status       string  `json:"status"`
	CustomStatus *string `json:"custom_status"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	Gender       *string `json:"gender"`
	Phone        *string `json:"phone"`
}
