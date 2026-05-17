package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/alakkaya/openscout/internal/repository"
	"github.com/google/uuid"
)

type Handler struct {
    userRepo   repository.UserRepository
    prefRepo   repository.UserPreferenceRepository
    feedbackRepo repository.FeedbackRepository
    log        *slog.Logger
}

func NewHandler(
    u repository.UserRepository,
    p repository.UserPreferenceRepository,
    f repository.FeedbackRepository,
    log *slog.Logger,
) *Handler {
    return &Handler{
        userRepo: u, prefRepo: p, feedbackRepo: f, log: log,
    }
}

// Request/Response types

type SubscribeRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type SubscribeResponse struct {
    ID    string `json:"id"`
    Email string `json:"email"`
}

type PreferencesRequest struct {
    UserID    string   `json:"user_id"`
    Languages []string `json:"languages"`
    Labels    []string `json:"labels"`
    MaxComplexity int `json:"max_complexity"`
}

type FeedbackRequest struct {
    UserID  string `json:"user_id"`
    IssueID string `json:"issue_id"`
    Status  string `json:"status"` // "solved" | "not_interested"
    Note    string `json:"note"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

// Subscribe creates a new user and default preferences.
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
    var req SubscribeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request")
        return
    }

    if req.Email == "" {
        writeError(w, http.StatusBadRequest, "email required")
        return
    }

    user := &domain.User{
        ID:        uuid.NewString(),
        Name:      req.Name,
        Email:     req.Email,
        Active:    true,
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
    }

    if err := h.userRepo.CreateUser(r.Context(), user); err != nil {
        h.log.Error("create user failed", "error", err)
        writeError(w, http.StatusInternalServerError, "failed to create user")
        return
    }

    // Create default preferences
    langs, err := json.Marshal([]string{"Go", "Python", "TypeScript"})
    if err != nil {
        h.log.Error("marshal default languages failed", "error", err)
        writeError(w, http.StatusInternalServerError, "failed to create default preferences")
        return
    }
    lbls, err := json.Marshal([]string{"good first issue"})
    if err != nil {
        h.log.Error("marshal default labels failed", "error", err)
        writeError(w, http.StatusInternalServerError, "failed to create default preferences")
        return
    }
    pref := &domain.UserPreference{
        UserID:        user.ID,
        Languages:     string(langs),
        Labels:        string(lbls),
        MaxComplexity: 5,
        CreatedAt:     time.Now().UTC(),
        UpdatedAt:     time.Now().UTC(),
    }
    if err := h.prefRepo.CreateOrUpdatePreference(r.Context(), pref); err != nil {
        h.log.Error("create default preferences failed", "error", err)
        writeError(w, http.StatusInternalServerError, "failed to create default preferences")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(SubscribeResponse{ID: user.ID, Email: user.Email})
}

// Preferences updates user language/label preferences.
func (h *Handler) Preferences(w http.ResponseWriter, r *http.Request) {
    var req PreferencesRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request")
        return
    }

    if req.UserID == "" {
        writeError(w, http.StatusBadRequest, "user_id required")
        return
    }

    langs, _ := json.Marshal(req.Languages)
    lbls, _ := json.Marshal(req.Labels)

    pref := &domain.UserPreference{
        UserID:        req.UserID,
        Languages:     string(langs),
        Labels:        string(lbls),
        MaxComplexity: req.MaxComplexity,
        CreatedAt:     time.Now().UTC(),
        UpdatedAt:     time.Now().UTC(),
    }

    if err := h.prefRepo.CreateOrUpdatePreference(r.Context(), pref); err != nil {
        h.log.Error("update preferences failed", "error", err)
        writeError(w, http.StatusInternalServerError, "failed to update preferences")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// Feedback records user action (solved/not_interested) on an issue.
func (h *Handler) Feedback(w http.ResponseWriter, r *http.Request) {
    var req FeedbackRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request")
        return
    }

    if req.UserID == "" || req.IssueID == "" || req.Status == "" {
        writeError(w, http.StatusBadRequest, "user_id, issue_id, and status required")
        return
    }

    if req.Status != "solved" && req.Status != "not_interested" {
        writeError(w, http.StatusBadRequest, "status must be 'solved' or 'not_interested'")
        return
    }

    if err := h.feedbackRepo.SaveFeedback(r.Context(), req.UserID, req.IssueID, req.Status); err != nil {
        h.log.Error("save feedback failed", "error", err)
        writeError(w, http.StatusInternalServerError, "failed to save feedback")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

func writeError(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}