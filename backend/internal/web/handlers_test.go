package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/alakkaya/openscout/internal/repository"
)

type stubUserRepo struct {
	createFn func(ctx context.Context, user *domain.User) error
}

func (s *stubUserRepo) CreateUser(ctx context.Context, user *domain.User) error {
	return s.createFn(ctx, user)
}

func (s *stubUserRepo) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepo) GetAllActiveUsers(ctx context.Context) ([]*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepo) UpdateUser(ctx context.Context, user *domain.User) error {
	return errors.New("not implemented")
}

func (s *stubUserRepo) DeactivateUser(ctx context.Context, userID string) error {
	return errors.New("not implemented")
}

type stubPrefRepo struct {
	createFn func(ctx context.Context, pref *domain.UserPreference) error
}

func (s *stubPrefRepo) CreateOrUpdatePreference(ctx context.Context, pref *domain.UserPreference) error {
	return s.createFn(ctx, pref)
}

func (s *stubPrefRepo) GetPreferenceByUserID(ctx context.Context, userID string) (*domain.UserPreference, error) {
	return nil, errors.New("not implemented")
}

type stubFeedbackRepo struct{}

func (s *stubFeedbackRepo) SaveFeedback(ctx context.Context, userID, issueID, feedback string) error {
	return errors.New("not implemented")
}

func (s *stubFeedbackRepo) HasUserRespondedToIssue(ctx context.Context, userID, issueID string) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *stubFeedbackRepo) GetUserFeedback(ctx context.Context, userID, issueID string) (*domain.UserIssueFeedback, error) {
	return nil, errors.New("not implemented")
}

var _ repository.UserRepository = (*stubUserRepo)(nil)
var _ repository.UserPreferenceRepository = (*stubPrefRepo)(nil)
var _ repository.FeedbackRepository = (*stubFeedbackRepo)(nil)

func TestSubscribeReturns500WhenDefaultPreferencesFail(t *testing.T) {
	t.Parallel()

	userRepo := &stubUserRepo{
		createFn: func(ctx context.Context, user *domain.User) error {
			return nil
		},
	}
	prefRepo := &stubPrefRepo{
		createFn: func(ctx context.Context, pref *domain.UserPreference) error {
			return errors.New("boom")
		},
	}
	h := &Handler{
		userRepo:     userRepo,
		prefRepo:     prefRepo,
		feedbackRepo: &stubFeedbackRepo{},
		log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	req := httptest.NewRequest(http.MethodPost, "/subscribe", strings.NewReader(`{"name":"Ada","email":"ada@example.com"}`))
	rec := httptest.NewRecorder()

	h.Subscribe(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "failed to create default preferences") {
		t.Fatalf("expected error body to mention default preferences, got %q", rec.Body.String())
	}
}