package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/retich-corp/user/internal/handler"
	"github.com/retich-corp/user/internal/repository"
	"github.com/retich-corp/user/internal/storage"

	_ "github.com/lib/pq"
)

// HealthResponse represente la reponse du endpoint /health.
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Initialise le backend de stockage des avatars.
	// Si les variables Azure sont presentes, utilise Azure Blob Storage.
	// Sinon, utilise le stockage local comme fallback pour le developpement.
	avatarStorage := initAvatarStorage(port)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Connected to database")

	userRepo := repository.NewUserRepository(db)
	userHandler := handler.NewUserHandler(userRepo, avatarStorage)

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/ready", readyHandler).Methods("GET")
	r.HandleFunc("/users", userHandler.ListUsers).Methods("GET")
	r.HandleFunc("/users/{id}", userHandler.GetProfile).Methods("GET")
	r.HandleFunc("/users/{id}", userHandler.UpdateProfile).Methods("PUT")
	r.HandleFunc("/users/{id}/avatar", userHandler.UpdateAvatar).Methods("PATCH")

	// Sert les fichiers statiques uniquement en mode stockage local.
	// En mode Azure Blob Storage, les fichiers sont servis directement par Azure.
	if uploadsDir := os.Getenv("UPLOADS_DIR"); uploadsDir != "" {
		r.PathPrefix("/uploads/").Handler(
			http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))),
		)
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("User Service starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// initAvatarStorage determine le backend de stockage des avatars en fonction
// des variables d'environnement configurees.
// Si AZURE_STORAGE_ACCOUNT_NAME, AZURE_STORAGE_ACCOUNT_KEY et
// AZURE_STORAGE_CONTAINER_NAME sont toutes definies, Azure Blob Storage est utilise.
// Sinon, le stockage local est utilise avec UPLOADS_DIR et BASE_URL.
func initAvatarStorage(port string) storage.AvatarStorage {
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT_KEY")
	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")

	// Si les trois variables Azure sont presentes, on utilise Azure Blob Storage.
	if accountName != "" && accountKey != "" && containerName != "" {
		azureStorage, err := storage.NewAzureBlobStorage(accountName, accountKey, containerName)
		if err != nil {
			log.Fatalf("Echec de l'initialisation d'Azure Blob Storage: %v", err)
		}
		log.Printf("Mode stockage : Azure Blob Storage (compte: %s, conteneur: %s)", accountName, containerName)
		return azureStorage
	}

	// Fallback : stockage local pour le developpement.
	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	// Cree le dossier d'uploads s'il n'existe pas encore.
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Fatalf("Echec de la creation du dossier uploads: %v", err)
	}

	log.Printf("Mode stockage : local (dossier: %s, URL de base: %s)", uploadsDir, baseURL)
	return storage.NewLocalStorage(uploadsDir, baseURL)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Service:   "user",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
