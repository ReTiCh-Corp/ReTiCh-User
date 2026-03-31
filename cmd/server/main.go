package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/retich-corp/user/internal/handler"
	"github.com/retich-corp/user/internal/repository"
	"github.com/retich-corp/user/internal/storage"

	_ "github.com/lib/pq"
)

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

	// Initialize avatar storage (Azure Blob in prod, local in dev)
	avatarStorage, uploadsDir := initAvatarStorage(port)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Connected to database")

	// AUTO_MIGRATE=true : exécute les migrations au démarrage (dev uniquement).
	if os.Getenv("AUTO_MIGRATE") == "true" {
		migrationsPath := os.Getenv("MIGRATIONS_PATH")
		if migrationsPath == "" {
			migrationsPath = "file://migrations"
		}
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Fatalf("Failed to create migration driver: %v", err)
		}
		m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
		if err != nil {
			log.Fatalf("Failed to initialize migrations: %v", err)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations applied successfully")
	}

	userRepo := repository.NewUserRepository(db)
	userHandler := handler.NewUserHandler(userRepo, avatarStorage)

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/ready", readyHandler).Methods("GET")
	r.HandleFunc("/users", userHandler.CreateUser).Methods("POST")
	r.HandleFunc("/users", userHandler.ListUsers).Methods("GET")
	r.HandleFunc("/users/check-username", userHandler.CheckUsername).Methods("GET")
	r.HandleFunc("/users/{id}", userHandler.GetProfile).Methods("GET")
	r.HandleFunc("/users/{id}", userHandler.UpdateProfile).Methods("PUT")
	r.HandleFunc("/users/{id}/onboarding", userHandler.CompleteOnboarding).Methods("PATCH")
	r.HandleFunc("/users/{id}/avatar", userHandler.UpdateAvatar).Methods("PATCH")

	// Serve static files only in local storage mode
	if uploadsDir != "" {
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

// initAvatarStorage picks Azure Blob Storage if env vars are set, otherwise local.
// Returns the storage backend and the uploads directory (empty string if Azure).
func initAvatarStorage(port string) (storage.AvatarStorage, string) {
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT_NAME")
	accountKey := os.Getenv("AZURE_STORAGE_ACCOUNT_KEY")
	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")

	if accountName != "" && accountKey != "" && containerName != "" {
		azureStorage, err := storage.NewAzureBlobStorage(accountName, accountKey, containerName)
		if err != nil {
			log.Fatalf("Failed to initialize Azure Blob Storage: %v", err)
		}
		log.Printf("Storage: Azure Blob Storage (account: %s, container: %s)", accountName, containerName)
		return azureStorage, ""
	}

	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	log.Printf("Storage: local (dir: %s, base URL: %s)", uploadsDir, baseURL)
	return storage.NewLocalStorage(uploadsDir, baseURL), uploadsDir
}
