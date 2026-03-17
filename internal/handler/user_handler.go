package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/retich-corp/user/internal/model"
	"github.com/retich-corp/user/internal/repository"
)

// maxUploadSize limite la taille des fichiers uploadés à 5 Mo.
const maxUploadSize = 5 * 1024 * 1024

// allowedMIMETypes liste les types d'image acceptés.
// La clé est le type MIME détecté depuis le contenu du fichier,
// la valeur est l'extension à utiliser pour le nom du fichier sur disque.
var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// userRepository définit les opérations en base attendues par le handler.
// Utiliser une interface permet d'injecter un mock dans les tests sans base de données réelle.
type userRepository interface {
	GetByID(id string) (*model.Profile, error)
	UpdateByID(id string, req *model.UpdateProfileRequest) (*model.Profile, error)
	UpdateAvatarURL(id, avatarURL string) (*model.Profile, error)
	List(search string, limit, offset int) ([]*model.Profile, int, error)
}

type listUsersResponse struct {
	Users  []*model.Profile `json:"users"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// UserHandler regroupe tous les handlers HTTP liés aux utilisateurs.
// uploadsDir : chemin du dossier où sont stockés les avatars sur disque.
// baseURL    : URL publique de base pour construire les liens vers les avatars.
type UserHandler struct {
	repo       userRepository
	uploadsDir string
	baseURL    string
}

// NewUserHandler crée un UserHandler avec les dépendances nécessaires.
// *repository.UserRepository satisfait userRepository implicitement (duck typing Go).
func NewUserHandler(repo userRepository, uploadsDir, baseURL string) *UserHandler {
	return &UserHandler{repo: repo, uploadsDir: uploadsDir, baseURL: baseURL}
}

// writeJSON factorise l'écriture d'une réponse JSON pour éviter la répétition
// du trio Header/WriteHeader/Encode dans chaque handler.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// UpdateAvatar gère PATCH /users/{id}/avatar.
// Attend un formulaire multipart avec un champ "avatar" contenant le fichier image.
// Le fichier est sauvegardé sous {uploadsDir}/{id}.{ext}, écrasant l'ancien avatar.
// Répond 200 + profil complet mis à jour, ou 400 / 404 / 500 selon le cas.
func (h *UserHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Limite le body à maxUploadSize pour rejeter les gros fichiers avant même de les lire.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large (max 5MB)"})
		return
	}

	// Récupère le fichier depuis le champ "avatar" du formulaire multipart.
	file, _, err := r.FormFile("avatar")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field 'avatar' is required"})
		return
	}
	defer file.Close()

	// Détecte le type MIME réel depuis les premiers octets du fichier.
	// On ne fait pas confiance au Content-Type envoyé par le client : n'importe qui
	// pourrait envoyer un exécutable en prétendant que c'est une image.
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

	// Remet le curseur au début du fichier pour pouvoir le lire en intégralité ensuite.
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// Nomme le fichier avec l'ID utilisateur : {id}.{ext}.
	// Cela garantit qu'un second upload remplace automatiquement l'ancien avatar
	// sans laisser de fichiers orphelins sur le disque.
	filename := id + ext
	dst, err := os.Create(filepath.Join(h.uploadsDir, filename))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	defer dst.Close()

	// Copie le fichier depuis la mémoire vers le disque.
	if _, err = io.Copy(dst, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// Construit l'URL publique de l'avatar et met à jour la base de données.
	avatarURL := h.baseURL + "/uploads/" + filename
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

// UpdateProfile gère PUT /users/{id}.
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

// ListUsers gère GET /users.
// Paramètres de requête :
//   - limit  : nombre de résultats par page (défaut 20, max 100)
//   - offset : décalage pour la pagination (défaut 0)
//   - q      : terme de recherche sur username et display_name (optionnel)
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

	search := q.Get("q")

	profiles, total, err := h.repo.List(search, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	if profiles == nil {
		profiles = []*model.Profile{}
	}

	writeJSON(w, http.StatusOK, listUsersResponse{
		Users:  profiles,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetProfile gère GET /users/{id}.
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
