package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/retich-corp/user/internal/model"
	"github.com/retich-corp/user/internal/repository"
	"github.com/retich-corp/user/internal/storage"
)

// maxUploadSize limite la taille des fichiers uploades a 5 Mo.
const maxUploadSize = 5 * 1024 * 1024

// allowedMIMETypes liste les types d'image acceptes.
// La cle est le type MIME detecte depuis le contenu du fichier,
// la valeur est l'extension a utiliser pour le nom du fichier stocke.
var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// userRepository definit les operations en base attendues par le handler.
// Utiliser une interface permet d'injecter un mock dans les tests sans base de donnees reelle.
type userRepository interface {
	GetByID(id string) (*model.Profile, error)
	UpdateByID(id string, req *model.UpdateProfileRequest) (*model.Profile, error)
	UpdateAvatarURL(id, avatarURL string) (*model.Profile, error)
	List(search, sort string, limit, offset int) ([]*model.UserSummary, int, error)
}

type paginationMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type listUsersResponse struct {
	Data       []*model.UserSummary `json:"data"`
	Pagination paginationMeta       `json:"pagination"`
}

// UserHandler regroupe tous les handlers HTTP lies aux utilisateurs.
// Il depend de l'interface userRepository pour l'acces aux donnees
// et de l'interface storage.AvatarStorage pour le stockage des avatars.
// Cette architecture respecte le principe d'inversion de dependance (SOLID).
type UserHandler struct {
	repo    userRepository
	storage storage.AvatarStorage
}

// NewUserHandler cree un UserHandler avec les dependances necessaires.
// repo : implementation du repository utilisateur (base de donnees).
// avatarStorage : implementation du stockage des avatars (local, Azure Blob, etc.).
func NewUserHandler(repo userRepository, avatarStorage storage.AvatarStorage) *UserHandler {
	return &UserHandler{repo: repo, storage: avatarStorage}
}

// writeJSON factorise l'ecriture d'une reponse JSON pour eviter la repetition
// du trio Header/WriteHeader/Encode dans chaque handler.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// UpdateAvatar gere PATCH /users/{id}/avatar.
// Attend un formulaire multipart avec un champ "avatar" contenant le fichier image.
// Le fichier est uploade via l'interface AvatarStorage (local ou Azure Blob Storage).
// Repond 200 + profil complet mis a jour, ou 400 / 404 / 500 selon le cas.
func (h *UserHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Limite le body a maxUploadSize pour rejeter les gros fichiers avant meme de les lire.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large (max 5MB)"})
		return
	}

	// Recupere le fichier depuis le champ "avatar" du formulaire multipart.
	file, _, err := r.FormFile("avatar")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field 'avatar' is required"})
		return
	}
	defer file.Close()

	// Detecte le type MIME reel depuis les premiers octets du fichier.
	// On ne fait pas confiance au Content-Type envoye par le client : n'importe qui
	// pourrait envoyer un executable en pretendant que c'est une image.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	mimeType := http.DetectContentType(buf[:n])
	ext, ok := allowedMIMETypes[mimeType]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported image type (jpeg, png, gif, webp only)"})
		return
	}

	// Remet le curseur au debut du fichier pour pouvoir le lire en integralite ensuite.
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// Delegue l'upload au service de stockage (local ou Azure Blob Storage).
	// L'interface AvatarStorage gere la logique specifique a chaque backend.
	avatarURL, err := h.storage.Upload(r.Context(), id, ext, file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// Met a jour l'URL de l'avatar en base de donnees.
	profile, err := h.repo.UpdateAvatarURL(id, avatarURL)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// UpdateProfile gere PUT /users/{id}.
// Remplace tous les champs modifiables du profil (remplacement complet).
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req model.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}

	profile, err := h.repo.UpdateByID(id, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// ListUsers gere GET /users.
// Parametres de requete :
//   - limit  : nombre de resultats par page (defaut 20, max 100)
//   - offset : decalage pour la pagination (defaut 0)
//   - search : terme de recherche sur username et email (optionnel, ILIKE)
//   - sort   : colonne de tri — username, -username, created_at, -created_at (defaut : username)
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if raw := q.Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		limit = v
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if raw := q.Get("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "offset must be a non-negative integer"})
			return
		}
		offset = v
	}

	search := q.Get("search")
	sort := q.Get("sort")

	users, total, err := h.repo.List(search, sort, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if users == nil {
		users = []*model.UserSummary{}
	}

	writeJSON(w, http.StatusOK, listUsersResponse{
		Data: users,
		Pagination: paginationMeta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// GetProfile gere GET /users/{id}.
// Retourne le profil complet de l'utilisateur ou 404 s'il n'existe pas.
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	profile, err := h.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, profile)
}
