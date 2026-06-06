package services

import (
	"context"

	"rize-api/internal/models"
	"rize-api/internal/repository"
)

type SessionService struct {
	sitRepo  *repository.SitRepository
	moveRepo *repository.MoveRepository
}

func NewSessionService(
	sitRepo *repository.SitRepository,
	moveRepo *repository.MoveRepository,
) *SessionService {
	return &SessionService{sitRepo: sitRepo, moveRepo: moveRepo}
}

// Sit

func (s *SessionService) StartSit(ctx context.Context, userID string) (*models.SitSession, error) {
	// end any existing active sit first
	s.sitRepo.End(ctx, userID) //nolint:errcheck
	return s.sitRepo.Start(ctx, userID)
}

func (s *SessionService) EndSit(ctx context.Context, userID string) (*models.SitSession, error) {
	return s.sitRepo.End(ctx, userID)
}

func (s *SessionService) GetActiveSit(ctx context.Context, userID string) (*models.SitSession, error) {
	return s.sitRepo.GetActive(ctx, userID)
}

func (s *SessionService) GetTodaySit(ctx context.Context, userID string) ([]*models.SitSession, error) {
	return s.sitRepo.GetToday(ctx, userID)
}

// Move

func (s *SessionService) LogMove(ctx context.Context, userID string, in *models.LogMoveInput) (*models.MoveSession, error) {
	// auto-close active sit when movement is logged
	s.sitRepo.End(ctx, userID) //nolint:errcheck
	return s.moveRepo.Log(ctx, userID, in)
}

func (s *SessionService) StartMove(ctx context.Context, userID string, in *models.StartMoveInput) (*models.MoveSession, error) {
	return s.moveRepo.Start(ctx, userID, in)
}

func (s *SessionService) EndMove(ctx context.Context, userID string, sessionID int64) (*models.MoveSession, error) {
	s.sitRepo.End(ctx, userID) //nolint:errcheck
	return s.moveRepo.End(ctx, userID, sessionID)
}

func (s *SessionService) GetTodayMove(ctx context.Context, userID string) ([]*models.MoveSession, error) {
	return s.moveRepo.GetToday(ctx, userID)
}

func (s *SessionService) DeleteMove(ctx context.Context, userID string, sessionID int64) (bool, error) {
	return s.moveRepo.Delete(ctx, userID, sessionID)
}
