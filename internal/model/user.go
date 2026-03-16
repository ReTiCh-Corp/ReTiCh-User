package model

import "time"

type Profile struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	DisplayName  *string    `json:"display_name"`
	AvatarURL    *string    `json:"avatar_url"`
	Bio          *string    `json:"bio"`
	Status       string     `json:"status"`
	CustomStatus *string    `json:"custom_status"`
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
}
