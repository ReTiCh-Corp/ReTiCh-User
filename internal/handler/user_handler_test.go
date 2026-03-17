package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/retich-corp/user/internal/model"
	"github.com/retich-corp/user/internal/repository"
)

// mockRepo implémente userRepository pour les tests sans base de données.
type mockRepo struct {
	profile *model.Profile
	err     error
}

func (m *mockRepo) GetByID(_ string) (*model.Profile, error) {
	return m.profile, m.err
}

func (m *mockRepo) UpdateByID(_ string, _ *model.UpdateProfileRequest) (*model.Profile, error) {
	return m.profile, m.err
}

func (m *mockRepo) UpdateAvatarURL(_, _ string) (*model.Profile, error) {
	return m.profile, m.err
}

func (m *mockRepo) List(_ string, _, _ int) ([]*model.Profile, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	if m.profile != nil {
		return []*model.Profile{m.profile}, 1, nil
	}
	return []*model.Profile{}, 0, nil
}

// sampleProfile retourne un profil de test réutilisable.
func sampleProfile() *model.Profile {
	name := "Alice Dupont"
	return &model.Profile{
		ID:          "test-id",
		Username:    "alice",
		DisplayName: &name,
		Status:      "online",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// newTestRouter configure un routeur mux pour que mux.Vars() soit renseigné dans les tests.
func newTestRouter(h *UserHandler) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/users", h.ListUsers).Methods("GET")
	r.HandleFunc("/users/{id}", h.GetProfile).Methods("GET")
	r.HandleFunc("/users/{id}", h.UpdateProfile).Methods("PUT")
	r.HandleFunc("/users/{id}/avatar", h.UpdateAvatar).Methods("PATCH")
	return r
}

// newAvatarRequest construit une requête multipart avec un fichier image simulé.
func newAvatarRequest(t *testing.T, userID string, content []byte, field string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, "avatar.png")
	if err != nil {
		t.Fatalf("could not create form file: %v", err)
	}
	part.Write(content)
	writer.Close()

	req := httptest.NewRequest("PATCH", "/users/"+userID+"/avatar", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// pngBytes retourne les octets de signature d'un fichier PNG valide.
// http.DetectContentType reconnaît les 8 premiers octets comme "image/png".
func pngBytes() []byte {
	return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
}

// =============================================================================
// GetProfile
// =============================================================================

func TestGetProfile_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/test-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var profile model.Profile
	if err := json.Unmarshal(w.Body.Bytes(), &profile); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if profile.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", profile.ID)
	}
}

func TestGetProfile_404(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: repository.ErrNotFound}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetProfile_500(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: errors.New("db error")}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/test-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// UpdateProfile
// =============================================================================

func TestUpdateProfile_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	body := `{"username":"alice","status":"online"}`
	req := httptest.NewRequest("PUT", "/users/test-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUpdateProfile_400_InvalidJSON(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("PUT", "/users/test-id", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateProfile_400_MissingUsername(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	body := `{"username":"","status":"online"}`
	req := httptest.NewRequest("PUT", "/users/test-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateProfile_404(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: repository.ErrNotFound}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	body := `{"username":"alice","status":"online"}`
	req := httptest.NewRequest("PUT", "/users/unknown", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// =============================================================================
// UpdateAvatar
// =============================================================================

func TestUpdateAvatar_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := newAvatarRequest(t, "test-id", pngBytes(), "avatar")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAvatar_400_MissingField(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	// Le champ s'appelle "file" au lieu de "avatar".
	req := newAvatarRequest(t, "test-id", pngBytes(), "file")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateAvatar_400_InvalidMIME(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := newAvatarRequest(t, "test-id", []byte("this is plain text"), "avatar")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateAvatar_404(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: repository.ErrNotFound}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := newAvatarRequest(t, "unknown", pngBytes(), "avatar")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// =============================================================================
// ListUsers
// =============================================================================

func TestListUsers_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp listUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(resp.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(resp.Users))
	}
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
	if resp.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", resp.Offset)
	}
}

func TestListUsers_WithSearch_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users?q=alice&limit=10&offset=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp listUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Limit)
	}
	if resp.Offset != 5 {
		t.Errorf("expected offset 5, got %d", resp.Offset)
	}
}

func TestListUsers_Empty_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp listUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Users == nil {
		t.Error("expected non-nil users slice, got nil")
	}
	if len(resp.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(resp.Users))
	}
}

func TestListUsers_InvalidLimit_400(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users?limit=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListUsers_NegativeOffset_400(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users?offset=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListUsers_500(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: errors.New("db error")}, t.TempDir(), "http://localhost:8083")
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
