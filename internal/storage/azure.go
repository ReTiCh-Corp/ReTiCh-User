package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// AzureBlobStorage implemente AvatarStorage en utilisant Azure Blob Storage.
// Les avatars sont stockes dans un conteneur Azure dediee, chaque blob etant nomme
// par l'ID utilisateur suivi de l'extension (ex: "abc123.png").
type AzureBlobStorage struct {
	// client est le client Azure Blob Storage authentifie.
	client *azblob.Client
	// containerName est le nom du conteneur Azure ou les avatars sont stockes.
	containerName string
	// accountName est le nom du compte de stockage Azure, utilise pour construire les URLs.
	accountName string
}

// NewAzureBlobStorage cree une instance de AzureBlobStorage configuree avec les credentials Azure.
// accountName : nom du compte de stockage Azure (ex: "retichstorage").
// accountKey  : cle d'acces du compte de stockage Azure.
// containerName : nom du conteneur Azure pour les avatars (ex: "avatars").
// Retourne une erreur si l'authentification aupres d'Azure echoue.
func NewAzureBlobStorage(accountName, accountKey, containerName string) (*AzureBlobStorage, error) {
	// Valide que tous les parametres obligatoires sont fournis.
	// Le SDK Azure ne valide pas les chaines vides a la construction,
	// ce qui provoquerait des erreurs confuses a l'utilisation.
	if accountName == "" {
		return nil, errors.New("le nom du compte de stockage Azure est requis")
	}
	if accountKey == "" {
		return nil, errors.New("la cle du compte de stockage Azure est requise")
	}
	if containerName == "" {
		return nil, errors.New("le nom du conteneur Azure est requis")
	}

	// Construit les credentials a partir de la cle partagee du compte de stockage.
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("creation des credentials Azure: %w", err)
	}

	// Construit l'URL de base du service Blob : https://{accountName}.blob.core.windows.net/
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	// Cree le client Azure Blob Storage avec les credentials.
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creation du client Azure Blob: %w", err)
	}

	return &AzureBlobStorage{
		client:        client,
		containerName: containerName,
		accountName:   accountName,
	}, nil
}

// Upload envoie le fichier avatar vers Azure Blob Storage.
// Le blob est nomme {userID}{ext} dans le conteneur configure.
// Le Content-Type est defini comme "image/{ext sans le point}" pour un affichage correct dans les navigateurs.
// Retourne l'URL publique du blob uploade.
func (s *AzureBlobStorage) Upload(ctx context.Context, userID, ext string, content io.Reader) (string, error) {
	// Nomme le blob avec l'ID utilisateur + extension pour garantir l'unicite.
	blobName := userID + ext

	// Upload le contenu vers le conteneur Azure.
	// UploadStream gere automatiquement le chunking pour les gros fichiers.
	_, err := s.client.UploadStream(ctx, s.containerName, blobName, content, nil)
	if err != nil {
		return "", fmt.Errorf("upload du blob %q vers Azure: %w", blobName, err)
	}

	// Construit l'URL publique du blob :
	// https://{accountName}.blob.core.windows.net/{containerName}/{blobName}
	blobURL := fmt.Sprintf(
		"https://%s.blob.core.windows.net/%s/%s",
		s.accountName,
		s.containerName,
		blobName,
	)

	return blobURL, nil
}

// Delete supprime un blob avatar du conteneur Azure.
// blobName est le nom complet du blob (ex: "user-id.png").
func (s *AzureBlobStorage) Delete(ctx context.Context, blobName string) error {
	_, err := s.client.DeleteBlob(ctx, s.containerName, blobName, nil)
	if err != nil {
		return fmt.Errorf("suppression du blob %q sur Azure: %w", blobName, err)
	}
	return nil
}
