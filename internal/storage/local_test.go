package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalStorage_Upload verifie que l'upload local ecrit le fichier sur disque
// et retourne l'URL publique correcte.
func TestLocalStorage_Upload(t *testing.T) {
	tmpDir := t.TempDir()
	baseURL := "http://localhost:8083"
	store := NewLocalStorage(tmpDir, baseURL)

	content := strings.NewReader("fake image data")
	url, err := store.Upload(context.Background(), "user-123", ".png", content)
	if err != nil {
		t.Fatalf("Upload ne devrait pas echouer: %v", err)
	}

	// Verifie que l'URL retournee est correcte.
	expectedURL := baseURL + "/uploads/user-123.png"
	if url != expectedURL {
		t.Errorf("URL attendue %q, obtenue %q", expectedURL, url)
	}

	// Verifie que le fichier existe sur le disque.
	filePath := filepath.Join(tmpDir, "user-123.png")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Le fichier devrait exister sur disque: %v", err)
	}
	if string(data) != "fake image data" {
		t.Errorf("Contenu attendu %q, obtenu %q", "fake image data", string(data))
	}
}

// TestLocalStorage_Upload_Overwrite verifie que l'upload ecrase un fichier existant.
func TestLocalStorage_Upload_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalStorage(tmpDir, "http://localhost:8083")

	// Premier upload.
	_, err := store.Upload(context.Background(), "user-123", ".png", strings.NewReader("first"))
	if err != nil {
		t.Fatalf("Premier upload echoue: %v", err)
	}

	// Deuxieme upload avec un contenu different.
	_, err = store.Upload(context.Background(), "user-123", ".png", strings.NewReader("second"))
	if err != nil {
		t.Fatalf("Deuxieme upload echoue: %v", err)
	}

	// Verifie que le fichier contient le deuxieme contenu.
	data, err := os.ReadFile(filepath.Join(tmpDir, "user-123.png"))
	if err != nil {
		t.Fatalf("Lecture du fichier echouee: %v", err)
	}
	if string(data) != "second" {
		t.Errorf("Contenu attendu %q, obtenu %q", "second", string(data))
	}
}

// TestLocalStorage_Delete verifie que la suppression retire le fichier du disque.
func TestLocalStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalStorage(tmpDir, "http://localhost:8083")

	// Cree un fichier a supprimer.
	filePath := filepath.Join(tmpDir, "user-123.png")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatalf("Creation du fichier echouee: %v", err)
	}

	// Supprime le fichier via le storage.
	err := store.Delete(context.Background(), "user-123.png")
	if err != nil {
		t.Fatalf("Delete ne devrait pas echouer: %v", err)
	}

	// Verifie que le fichier n'existe plus.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Le fichier devrait avoir ete supprime du disque")
	}
}

// TestLocalStorage_Delete_NonExistent verifie que la suppression d'un fichier
// inexistant ne retourne pas d'erreur (idempotence).
func TestLocalStorage_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalStorage(tmpDir, "http://localhost:8083")

	err := store.Delete(context.Background(), "inexistant.png")
	if err != nil {
		t.Errorf("Delete d'un fichier inexistant ne devrait pas echouer: %v", err)
	}
}

// TestLocalStorage_Upload_InvalidDir verifie que l'upload echoue
// si le dossier de destination n'existe pas.
func TestLocalStorage_Upload_InvalidDir(t *testing.T) {
	store := NewLocalStorage("/chemin/inexistant/total", "http://localhost:8083")

	_, err := store.Upload(context.Background(), "user-123", ".png", strings.NewReader("data"))
	if err == nil {
		t.Error("Upload devrait echouer si le dossier n'existe pas")
	}
}
