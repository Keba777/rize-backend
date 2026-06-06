package services

import (
	"context"
	"fmt"

	"rize-api/internal/models"
	"rize-api/internal/repository"
	"rize-api/pkg/jwt"
	"rize-api/pkg/mailer"
)

type AuthService struct {
	userRepo      *repository.UserRepository
	mailer        *mailer.Mailer
	jwtSecret     string
	jwtTTLHours   int
	magicLinkTTL  int
	baseURL        string
}

func NewAuthService(
	userRepo *repository.UserRepository,
	m *mailer.Mailer,
	jwtSecret string,
	jwtTTLHours int,
	magicLinkTTL int,
	baseURL string,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		mailer:       m,
		jwtSecret:    jwtSecret,
		jwtTTLHours:  jwtTTLHours,
		magicLinkTTL: magicLinkTTL,
		baseURL:      baseURL,
	}
}

func (s *AuthService) SendMagicLink(ctx context.Context, email string) error {
	user, err := s.userRepo.FindOrCreate(ctx, email)
	if err != nil {
		return err
	}
	token, err := s.userRepo.CreateMagicLink(ctx, user.ID, s.magicLinkTTL)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/auth/verify?token=%s", s.baseURL, token)
	return s.mailer.SendMagicLink(email, link)
}

func (s *AuthService) VerifyMagicLink(ctx context.Context, token string) (string, *models.User, error) {
	userID, err := s.userRepo.ConsumeMagicLink(ctx, token)
	if err != nil {
		return "", nil, err
	}
	jwtToken, err := jwt.Issue(userID, s.jwtSecret, s.jwtTTLHours)
	if err != nil {
		return "", nil, err
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	return jwtToken, user, err
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}
