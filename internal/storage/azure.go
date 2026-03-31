package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// AzureBlobStorage stores avatars in Azure Blob Storage.
type AzureBlobStorage struct {
	client        *azblob.Client
	containerName string
	accountName   string
}

func NewAzureBlobStorage(accountName, accountKey, containerName string) (*AzureBlobStorage, error) {
	if accountName == "" {
		return nil, errors.New("Azure storage account name is required")
	}
	if accountKey == "" {
		return nil, errors.New("Azure storage account key is required")
	}
	if containerName == "" {
		return nil, errors.New("Azure storage container name is required")
	}

	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("create Azure credentials: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create Azure Blob client: %w", err)
	}

	return &AzureBlobStorage{
		client:        client,
		containerName: containerName,
		accountName:   accountName,
	}, nil
}

func (s *AzureBlobStorage) Upload(ctx context.Context, userID, ext string, content io.Reader) (string, error) {
	blobName := userID + ext

	_, err := s.client.UploadStream(ctx, s.containerName, blobName, content, nil)
	if err != nil {
		return "", fmt.Errorf("upload blob %q to Azure: %w", blobName, err)
	}

	blobURL := fmt.Sprintf(
		"https://%s.blob.core.windows.net/%s/%s",
		s.accountName,
		s.containerName,
		blobName,
	)
	return blobURL, nil
}

func (s *AzureBlobStorage) Delete(ctx context.Context, blobName string) error {
	_, err := s.client.DeleteBlob(ctx, s.containerName, blobName, nil)
	if err != nil {
		return fmt.Errorf("delete blob %q on Azure: %w", blobName, err)
	}
	return nil
}
