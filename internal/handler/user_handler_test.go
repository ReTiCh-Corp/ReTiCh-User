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
	"github.com/retich-corp/user/internal/storage"
)

// testStorage creates a local storage for tests.
func testStorage(t *testing.T) storage.AvatarStorage {
	t.Helper()
	return storage.NewLocalStorage(t.TempDir(), "http://localhost:8083")
}

// mockRepo implémente userRepository pour les tests sans base de données.
type mockRepo struct {
	profile         *model.Profile
	user            *model.User
	err             error
	usernameProfile *model.Profile
	usernameErr     error
}

func (m *mockRepo) EnsureUserAndProfile(_, _ string) error {
	return nil
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

func (m *mockRepo) List(_, _ string, _, _ int) ([]*model.UserSummary, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	if m.profile != nil {
		username := ""
		if m.profile.Username != nil {
			username = *m.profile.Username
		}
		return []*model.UserSummary{{ID: m.profile.ID, Email: "alice@example.com", Username: username}}, 1, nil
	}
	return []*model.UserSummary{}, 0, nil
}

func (m *mockRepo) Create(id, email string) (*model.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &model.User{
		ID:                  id,
		Email:               email,
		OnboardingCompleted: false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}, nil
}

func (m *mockRepo) GetByUsername(_ string) (*model.Profile, error) {
	if m.usernameProfile != nil {
		return m.usernameProfile, nil
	}
	if m.usernameErr != nil {
		return nil, m.usernameErr
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepo) GetByEmail(_ string) (*model.User, error) {
	if m.user != nil {
		return m.user, nil
	}
	if m.err != nil {
		return nil, m.err
	}
	return nil, repository.ErrNotFound
}

func (m *mockRepo) CompleteOnboarding(_ string) error {
	return m.err
}

// sampleProfile retourne un profil de test réutilisable.
func sampleProfile() *model.Profile {
	name := "Alice Dupont"
	username := "alice"
	return &model.Profile{
		ID:          "test-id",
		Username:    &username,
		DisplayName: &name,
		Status:      "online",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// newTestRouter configure un routeur mux pour que mux.Vars() soit renseigné dans les tests.
func newTestRouter(h *UserHandler) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/users", h.CreateUser).Methods("POST")
	r.HandleFunc("/users", h.ListUsers).Methods("GET")
	r.HandleFunc("/users/check-username", h.CheckUsername).Methods("GET")
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
// CreateUser
// =============================================================================

func TestCreateUser_201_NewUser(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	body := `{"id":"new-uuid","email":"new@example.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp model.CreateUserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if !resp.IsNewUser {
		t.Error("expected is_new_user to be true")
	}
	if resp.OnboardingCompleted {
		t.Error("expected onboarding_completed to be false")
	}
}

func TestCreateUser_200_ExistingUser(t *testing.T) {
	existing := &model.User{
		ID:                  "existing-uuid",
		Email:               "existing@example.com",
		OnboardingCompleted: false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	h := NewUserHandler(&mockRepo{user: existing}, testStorage(t))
	r := newTestRouter(h)

	body := `{"id":"new-uuid","email":"existing@example.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp model.CreateUserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.IsNewUser {
		t.Error("expected is_new_user to be false")
	}
	if resp.ID != "existing-uuid" {
		t.Errorf("expected ID 'existing-uuid', got %q", resp.ID)
	}
}

func TestCreateUser_400_MissingID(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_400_MissingEmail(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	body := `{"id":"some-uuid"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_400_InvalidJSON(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("POST", "/users", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_500_DBError(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: errors.New("db error")}, testStorage(t))
	r := newTestRouter(h)

	body := `{"id":"new-uuid","email":"new@example.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// GetProfile
// =============================================================================

func TestGetProfile_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
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
	h := NewUserHandler(&mockRepo{err: repository.ErrNotFound}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetProfile_500(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: errors.New("db error")}, testStorage(t))
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
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
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
	h := NewUserHandler(&mockRepo{}, testStorage(t))
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
	h := NewUserHandler(&mockRepo{}, testStorage(t))
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

func TestUpdateProfile_500(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: errors.New("db error")}, testStorage(t))
	r := newTestRouter(h)

	body := `{"username":"alice","status":"online"}`
	req := httptest.NewRequest("PUT", "/users/test-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestUpdateProfile_404(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: repository.ErrNotFound}, testStorage(t))
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
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
	r := newTestRouter(h)

	req := newAvatarRequest(t, "test-id", pngBytes(), "avatar")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAvatar_400_MissingField(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
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
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
	r := newTestRouter(h)

	req := newAvatarRequest(t, "test-id", []byte("this is plain text"), "avatar")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateAvatar_404(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: repository.ErrNotFound}, testStorage(t))
	r := newTestRouter(h)

	req := newAvatarRequest(t, "unknown", pngBytes(), "avatar")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateAvatar_500(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: errors.New("db error")}, testStorage(t))
	r := newTestRouter(h)

	req := newAvatarRequest(t, "test-id", pngBytes(), "avatar")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// ListUsers
// =============================================================================

func TestListUsers_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
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
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 user, got %d", len(resp.Data))
	}
	if resp.Pagination.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Pagination.Total)
	}
	if resp.Pagination.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", resp.Pagination.Limit)
	}
	if resp.Pagination.Offset != 0 {
		t.Errorf("expected default offset 0, got %d", resp.Pagination.Offset)
	}
}

func TestListUsers_WithSearch_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users?search=alice&sort=-username&limit=10&offset=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp listUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Pagination.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Pagination.Limit)
	}
	if resp.Pagination.Offset != 5 {
		t.Errorf("expected offset 5, got %d", resp.Pagination.Offset)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 user in data, got %d", len(resp.Data))
	}
}

func TestListUsers_Empty_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
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
	if resp.Data == nil {
		t.Error("expected non-nil data slice, got nil")
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 users, got %d", len(resp.Data))
	}
}

func TestListUsers_LimitCap_200(t *testing.T) {
	h := NewUserHandler(&mockRepo{profile: sampleProfile()}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users?limit=200", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp listUsersResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Pagination.Limit != 100 {
		t.Errorf("expected limit capped at 100, got %d", resp.Pagination.Limit)
	}
}

func TestListUsers_InvalidLimit_400(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users?limit=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListUsers_NegativeOffset_400(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users?offset=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListUsers_500(t *testing.T) {
	h := NewUserHandler(&mockRepo{err: errors.New("db error")}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// =============================================================================
// CheckUsername
// =============================================================================

func TestCheckUsername_Available(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/check-username?username=new_user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if !resp["available"] {
		t.Error("expected available to be true")
	}
}

func TestCheckUsername_Taken(t *testing.T) {
	h := NewUserHandler(&mockRepo{usernameProfile: sampleProfile()}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/check-username?username=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp["available"] {
		t.Error("expected available to be false")
	}
}

func TestCheckUsername_MissingParam(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/check-username", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCheckUsername_InvalidFormat(t *testing.T) {
	h := NewUserHandler(&mockRepo{}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/check-username?username=ab", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCheckUsername_500(t *testing.T) {
	h := NewUserHandler(&mockRepo{usernameErr: errors.New("db error")}, testStorage(t))
	r := newTestRouter(h)

	req := httptest.NewRequest("GET", "/users/check-username?username=some_user", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
