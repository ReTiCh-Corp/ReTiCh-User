package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/retich-corp/user/internal/model"
)

// columns liste les colonnes retournées par toutes les requêtes du repository.
var columns = []string{
	"id", "username", "display_name", "avatar_url",
	"bio", "status", "custom_status", "last_seen_at",
	"created_at", "updated_at",
}

// newMock initialise un *sql.DB mockée et le Sqlmock associé.
func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	return db, mock
}

// sampleRow retourne une ligne de résultat valide pour les tests.
func sampleRow() *sqlmock.Rows {
	now := time.Now()
	name := "Alice"
	return sqlmock.NewRows(columns).AddRow(
		"test-id", "alice", &name, nil,
		nil, "online", nil, nil,
		now, now,
	)
}

// =============================================================================
// GetByID
// =============================================================================

func TestGetByID_Found(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM profiles WHERE id").
		WithArgs("test-id").
		WillReturnRows(sampleRow())

	repo := NewUserRepository(db)
	profile, err := repo.GetByID("test-id")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", profile.ID)
	}
	if profile.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", profile.Username)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM profiles WHERE id").
		WithArgs("unknown").
		WillReturnError(sql.ErrNoRows)

	repo := NewUserRepository(db)
	_, err := repo.GetByID("unknown")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByID_DBError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM profiles WHERE id").
		WithArgs("test-id").
		WillReturnError(errors.New("connection lost"))

	repo := NewUserRepository(db)
	_, err := repo.GetByID("test-id")

	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a db error, got %v", err)
	}
}

// =============================================================================
// UpdateByID
// =============================================================================

func TestUpdateByID_Found(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("UPDATE profiles").
		WithArgs("test-id", "alice_updated", nil, nil, nil, "busy", nil).
		WillReturnRows(sampleRow())

	repo := NewUserRepository(db)
	req := &model.UpdateProfileRequest{
		Username: "alice_updated",
		Status:   "busy",
	}
	profile, err := repo.UpdateByID("test-id", req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile == nil {
		t.Fatal("expected a profile, got nil")
	}
}

func TestUpdateByID_NotFound(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("UPDATE profiles").
		WithArgs("unknown", "alice", nil, nil, nil, "online", nil).
		WillReturnError(sql.ErrNoRows)

	repo := NewUserRepository(db)
	req := &model.UpdateProfileRequest{Username: "alice", Status: "online"}
	_, err := repo.UpdateByID("unknown", req)

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// =============================================================================
// UpdateAvatarURL
// =============================================================================

func TestUpdateAvatarURL_Found(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	avatarURL := "http://localhost:8083/uploads/test-id.png"
	mock.ExpectQuery("UPDATE profiles").
		WithArgs("test-id", avatarURL).
		WillReturnRows(sampleRow())

	repo := NewUserRepository(db)
	profile, err := repo.UpdateAvatarURL("test-id", avatarURL)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile == nil {
		t.Fatal("expected a profile, got nil")
	}
}

func TestUpdateAvatarURL_NotFound(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("UPDATE profiles").
		WithArgs("unknown", "http://localhost:8083/uploads/unknown.png").
		WillReturnError(sql.ErrNoRows)

	repo := NewUserRepository(db)
	_, err := repo.UpdateAvatarURL("unknown", "http://localhost:8083/uploads/unknown.png")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
