package storage

import (
	"testing"
)

// TestNewAzureBlobStorage_EmptyParams verifie que la creation du client Azure
// echoue proprement lorsqu'un parametre obligatoire est vide.
func TestNewAzureBlobStorage_EmptyParams(t *testing.T) {
	tests := []struct {
		name          string
		accountName   string
		accountKey    string
		containerName string
	}{
		{
			name:          "cle de compte vide",
			accountName:   "testaccount",
			accountKey:    "",
			containerName: "avatars",
		},
		{
			name:          "nom de compte vide",
			accountName:   "",
			accountKey:    "dGVzdGtleQ==",
			containerName: "avatars",
		},
		{
			name:          "nom de conteneur vide",
			accountName:   "testaccount",
			accountKey:    "dGVzdGtleQ==",
			containerName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAzureBlobStorage(tt.accountName, tt.accountKey, tt.containerName)
			if err == nil {
				t.Errorf("NewAzureBlobStorage devrait echouer avec %s", tt.name)
			}
		})
	}
}

// TestNewAzureBlobStorage_ValidCredentials verifie que la creation du client Azure
// reussit avec des credentials au format valide.
// Note : ce test ne contacte pas Azure, il verifie uniquement l'initialisation du client.
func TestNewAzureBlobStorage_ValidCredentials(t *testing.T) {
	// Utilise une cle de test encodee en base64 (format attendu par Azure).
	// Cette cle est fictive et n'a aucune valeur de securite.
	fakeKey := "dGVzdGtleXRlc3RrZXl0ZXN0a2V5dGVzdGtleXRlc3Q="
	store, err := NewAzureBlobStorage("testaccount", fakeKey, "avatars")
	if err != nil {
		t.Fatalf("NewAzureBlobStorage ne devrait pas echouer avec des credentials valides: %v", err)
	}
	if store == nil {
		t.Fatal("Le storage retourne ne devrait pas etre nil")
	}
	if store.containerName != "avatars" {
		t.Errorf("containerName attendu %q, obtenu %q", "avatars", store.containerName)
	}
	if store.accountName != "testaccount" {
		t.Errorf("accountName attendu %q, obtenu %q", "testaccount", store.accountName)
	}
}

// TestAzureBlobStorage_ImplementsInterface verifie a la compilation que
// AzureBlobStorage satisfait bien l'interface AvatarStorage.
// Cette assertion de type empeche une regression si l'interface est modifiee.
func TestAzureBlobStorage_ImplementsInterface(t *testing.T) {
	var _ AvatarStorage = (*AzureBlobStorage)(nil)
}

// TestLocalStorage_ImplementsInterface verifie a la compilation que
// LocalStorage satisfait bien l'interface AvatarStorage.
func TestLocalStorage_ImplementsInterface(t *testing.T) {
	var _ AvatarStorage = (*LocalStorage)(nil)
}
