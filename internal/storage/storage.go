package storage

import (
	"context"
	"io"
)

// AvatarStorage defines the interface for storing user avatar images.
// Implementations include local filesystem (dev) and Azure Blob Storage (prod).
type AvatarStorage interface {
	// Upload saves the avatar file and returns the public URL.
	// userID is used as the blob/file name prefix.
	// ext is the file extension (e.g. ".png", ".jpg").
	Upload(ctx context.Context, userID, ext string, content io.Reader) (string, error)

	// Delete removes the avatar from storage.
	// blobName is the full filename (e.g. "user-id.png").
	Delete(ctx context.Context, blobName string) error
}
