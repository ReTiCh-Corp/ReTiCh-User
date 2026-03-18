// Package storage definit l'abstraction pour le stockage des avatars utilisateurs.
// Ce package applique le principe d'inversion de dependance (SOLID) :
// le handler depend de l'interface AvatarStorage, pas d'une implementation concrete.
// Cela permet de basculer entre stockage local et Azure Blob Storage sans modifier le handler.
package storage

import (
	"context"
	"io"
)

// AvatarStorage definit les operations de stockage pour les avatars utilisateurs.
// Chaque implementation (locale, Azure Blob, S3, etc.) doit satisfaire cette interface.
type AvatarStorage interface {
	// Upload envoie le contenu du fichier avatar vers le stockage distant ou local.
	// userID identifie l'utilisateur proprietaire de l'avatar.
	// ext est l'extension du fichier (ex: ".png", ".jpg").
	// content est le flux de donnees du fichier a uploader.
	// Retourne l'URL publique de l'avatar uploade, ou une erreur.
	Upload(ctx context.Context, userID, ext string, content io.Reader) (string, error)

	// Delete supprime l'avatar d'un utilisateur du stockage.
	// blobName est le nom du fichier/blob a supprimer (ex: "user-id.png").
	// Retourne une erreur si la suppression echoue.
	Delete(ctx context.Context, blobName string) error
}
