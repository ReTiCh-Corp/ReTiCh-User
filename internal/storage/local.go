package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implemente AvatarStorage en ecrivant les fichiers sur le disque local.
// Cette implementation est utile pour le developpement local et les tests d'integration.
type LocalStorage struct {
	// uploadsDir est le chemin du dossier ou les avatars sont enregistres sur disque.
	uploadsDir string
	// baseURL est l'URL publique de base pour construire les liens vers les avatars.
	baseURL string
}

// NewLocalStorage cree une instance de LocalStorage.
// uploadsDir : chemin du repertoire de stockage local.
// baseURL : URL de base pour construire les URLs publiques des avatars.
func NewLocalStorage(uploadsDir, baseURL string) *LocalStorage {
	return &LocalStorage{
		uploadsDir: uploadsDir,
		baseURL:    baseURL,
	}
}

// Upload ecrit le fichier avatar sur le disque local dans le dossier uploadsDir.
// Le fichier est nomme {userID}{ext} pour garantir l'unicite et ecraser l'ancien avatar.
// Retourne l'URL publique du fichier enregistre.
func (s *LocalStorage) Upload(_ context.Context, userID, ext string, content io.Reader) (string, error) {
	filename := userID + ext
	dstPath := filepath.Join(s.uploadsDir, filename)

	// Cree le fichier destination sur le disque. Si le fichier existe deja, il est ecrase.
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("creation du fichier avatar: %w", err)
	}
	defer dst.Close()

	// Copie le contenu du flux source vers le fichier destination.
	if _, err = io.Copy(dst, content); err != nil {
		return "", fmt.Errorf("ecriture du fichier avatar: %w", err)
	}

	// Construit l'URL publique : {baseURL}/uploads/{filename}
	avatarURL := s.baseURL + "/uploads/" + filename
	return avatarURL, nil
}

// Delete supprime le fichier avatar du disque local.
// Si le fichier n'existe pas, l'erreur est ignoree (idempotence).
func (s *LocalStorage) Delete(_ context.Context, blobName string) error {
	dstPath := filepath.Join(s.uploadsDir, blobName)
	err := os.Remove(dstPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("suppression du fichier avatar: %w", err)
	}
	return nil
}
