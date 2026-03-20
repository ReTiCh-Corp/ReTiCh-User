package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/retich-corp/user/internal/model"
)

// columns liste les colonnes retournées par les requêtes profile du repository.
var columns = []string{
	"id", "username", "display_name", "avatar_url",
	"bio", "status", "custom_status",
	"first_name", "last_name", "gender", "phone",
	"last_seen_at", "created_at", "updated_at",
}

// userColumns liste les colonnes retournées par les requêtes user du repository.
var userColumns = []string{"id", "email", "onboarding_completed", "created_at", "updated_at"}

// newMock initialise un *sql.DB mockée et le Sqlmock associé.
func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	return db, mock
}

// sampleRow retourne une ligne de résultat valide pour les tests profile.
func sampleRow() *sqlmock.Rows {
	now := time.Now()
	name := "Alice"
	username := "alice"
	return sqlmock.NewRows(columns).AddRow(
		"test-id", &username, &name, nil,
		nil, "online", nil,
		nil, nil, nil, nil,
		nil, now, now,
	)
}

// sampleUserRow retourne une ligne de résultat valide pour les tests user.
func sampleUserRow() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows(userColumns).AddRow(
		"test-id", "alice@example.com", false, now, now,
	)
}

// =============================================================================
// Create
// =============================================================================

func TestCreate_Success(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("new-uuid", "new@example.com").
		WillReturnRows(sampleUserRow())

	repo := NewUserRepository(db)
	user, err := repo.Create("new-uuid", "new@example.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", user.ID)
	}
	if user.OnboardingCompleted {
		t.Error("expected onboarding_completed to be false")
	}
}

func TestCreate_DBError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("new-uuid", "new@example.com").
		WillReturnError(errors.New("duplicate key"))

	repo := NewUserRepository(db)
	_, err := repo.Create("new-uuid", "new@example.com")

	if err == nil {
		t.Error("expected an error, got nil")
	}
}

// =============================================================================
// GetByEmail
// =============================================================================

func TestGetByEmail_Found(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("alice@example.com").
		WillReturnRows(sampleUserRow())

	repo := NewUserRepository(db)
	user, err := repo.GetByEmail("alice@example.com")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", user.Email)
	}
}

func TestGetByEmail_NotFound(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("unknown@example.com").
		WillReturnError(sql.ErrNoRows)

	repo := NewUserRepository(db)
	_, err := repo.GetByEmail("unknown@example.com")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByEmail_DBError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("alice@example.com").
		WillReturnError(errors.New("connection lost"))

	repo := NewUserRepository(db)
	_, err := repo.GetByEmail("alice@example.com")

	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a db error, got %v", err)
	}
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
	if profile.Username == nil || *profile.Username != "alice" {
		t.Errorf("expected username 'alice', got %v", profile.Username)
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
		WithArgs("test-id", "alice_updated", nil, nil, nil, "busy", nil, nil, nil, nil, nil).
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

func TestUpdateByID_DBError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("UPDATE profiles").
		WithArgs("test-id", "alice", nil, nil, nil, "online", nil, nil, nil, nil, nil).
		WillReturnError(errors.New("connection lost"))

	repo := NewUserRepository(db)
	req := &model.UpdateProfileRequest{Username: "alice", Status: "online"}
	_, err := repo.UpdateByID("test-id", req)

	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a db error, got %v", err)
	}
}

func TestUpdateByID_NotFound(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("UPDATE profiles").
		WithArgs("unknown", "alice", nil, nil, nil, "online", nil, nil, nil, nil, nil).
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

func TestUpdateAvatarURL_DBError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery("UPDATE profiles").
		WithArgs("test-id", "http://localhost:8083/uploads/test-id.png").
		WillReturnError(errors.New("connection lost"))

	repo := NewUserRepository(db)
	_, err := repo.UpdateAvatarURL("test-id", "http://localhost:8083/uploads/test-id.png")

	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a db error, got %v", err)
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

// =============================================================================
// List
// =============================================================================

// summaryColumns are the columns returned by the List query.
var summaryColumns = []string{"id", "email", "username", "onboarding_completed"}

// sampleSummaryRow returns a single UserSummary row for tests.
func sampleSummaryRow() *sqlmock.Rows {
	return sqlmock.NewRows(summaryColumns).AddRow("test-id", "alice@example.com", "alice", false)
}

func TestList_NoSearch(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT (.+) FROM users").
		WillReturnRows(sampleSummaryRow())

	repo := NewUserRepository(db)
	users, total, err := repo.List("", "", 20, 0)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
	if users[0].Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", users[0].Email)
	}
}

func TestList_WithSearch(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT (.+) FROM users").
		WillReturnRows(sampleSummaryRow())

	repo := NewUserRepository(db)
	users, total, err := repo.List("alice", "username", 20, 0)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestList_Empty(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT (.+) FROM users").
		WillReturnRows(sqlmock.NewRows(summaryColumns))

	repo := NewUserRepository(db)
	users, total, err := repo.List("", "-created_at", 20, 0)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestList_SelectQueryError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT (.+) FROM users").
		WillReturnError(errors.New("connection lost"))

	repo := NewUserRepository(db)
	_, _, err := repo.List("", "", 20, 0)

	if err == nil {
		t.Error("expected an error, got nil")
	}
}

func TestList_WithSearchCountError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnError(errors.New("connection lost"))

	repo := NewUserRepository(db)
	_, _, err := repo.List("alice", "", 20, 0)

	if err == nil {
		t.Error("expected an error, got nil")
	}
}

func TestList_CountError(t *testing.T) {
	db, mock := newMock(t)
	defer db.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnError(errors.New("connection lost"))

	repo := NewUserRepository(db)
	_, _, err := repo.List("", "", 20, 0)

	if err == nil {
		t.Error("expected an error, got nil")
	}
}
