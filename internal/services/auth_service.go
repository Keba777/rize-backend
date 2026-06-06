package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"rize-api/internal/models"
	"rize-api/internal/repository"
	"rize-api/pkg/jwt"
	"rize-api/pkg/mailer"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo           *repository.UserRepository
	mailer             *mailer.Mailer
	jwtSecret          string
	jwtTTLHours        int
	magicLinkTTL       int
	baseURL            string
	frontendURL        string
	googleClientID     string
	googleClientSecret string
	googleRedirectURL  string
}

func NewAuthService(
	userRepo *repository.UserRepository,
	m *mailer.Mailer,
	jwtSecret string,
	jwtTTLHours int,
	magicLinkTTL int,
	baseURL string,
	frontendURL string,
	googleClientID string,
	googleClientSecret string,
	googleRedirectURL string,
) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		mailer:             m,
		jwtSecret:          jwtSecret,
		jwtTTLHours:        jwtTTLHours,
		magicLinkTTL:       magicLinkTTL,
		baseURL:            baseURL,
		frontendURL:        frontendURL,
		googleClientID:     googleClientID,
		googleClientSecret: googleClientSecret,
		googleRedirectURL:  googleRedirectURL,
	}
}

// --- Magic link (existing) ---

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

// --- Email + password ---

func (s *AuthService) Register(ctx context.Context, email, password string) error {
	existing, _, _, err := s.userRepo.FindByEmailWithAuth(ctx, email)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	user, err := s.userRepo.CreateWithPassword(ctx, email, string(hash))
	if err != nil {
		return err
	}

	token, err := s.userRepo.CreateEmailVerifyToken(ctx, user.ID, 24*60) // 24 hours
	if err != nil {
		return err
	}

	link := fmt.Sprintf("%s/auth/verify-email?token=%s", s.frontendURL, token)
	return s.mailer.SendVerificationEmail(email, link)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	user, hash, verified, err := s.userRepo.FindByEmailWithAuth(ctx, email)
	if err != nil {
		return "", nil, err
	}
	if user == nil || hash == nil {
		return "", nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*hash), []byte(password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}
	if !verified {
		return "", nil, errors.New("email not verified — check your inbox")
	}

	token, err := jwt.Issue(user.ID, s.jwtSecret, s.jwtTTLHours)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) ResendVerification(ctx context.Context, email string) error {
	user, _, verified, err := s.userRepo.FindByEmailWithAuth(ctx, email)
	if err != nil || user == nil || verified {
		return nil // silent — don't reveal whether email exists or is verified
	}
	token, err := s.userRepo.CreateEmailVerifyToken(ctx, user.ID, 24*60)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/auth/verify-email?token=%s", s.frontendURL, token)
	return s.mailer.SendVerificationEmail(email, link)
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	userID, err := s.userRepo.ConsumeEmailVerifyToken(ctx, token)
	if err != nil {
		return err
	}
	return s.userRepo.SetEmailVerified(ctx, userID)
}

// --- Google OAuth ---

func (s *AuthService) GoogleAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", s.googleClientID)
	params.Set("redirect_uri", s.googleRedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("access_type", "online")
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

type googleTokenResp struct {
	AccessToken string `json:"access_token"`
}

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (s *AuthService) HandleGoogleCallback(ctx context.Context, code string) (string, *models.User, error) {
	// Exchange authorization code for access token
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", s.googleClientID)
	form.Set("client_secret", s.googleClientSecret)
	form.Set("redirect_uri", s.googleRedirectURL)
	form.Set("grant_type", "authorization_code")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", form)
	if err != nil {
		return "", nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tokenResp googleTokenResp
	if err := json.Unmarshal(body, &tokenResp); err != nil || tokenResp.AccessToken == "" {
		return "", nil, errors.New("failed to get access token from Google")
	}

	// Fetch user info
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	infoResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("get user info: %w", err)
	}
	defer infoResp.Body.Close()
	infoBody, _ := io.ReadAll(infoResp.Body)

	var info googleUserInfo
	if err := json.Unmarshal(infoBody, &info); err != nil || info.Email == "" {
		return "", nil, errors.New("failed to get user info from Google")
	}

	var name *string
	if info.Name != "" {
		name = &info.Name
	}

	user, err := s.userRepo.FindOrCreateByGoogle(ctx, info.ID, info.Email, name)
	if err != nil {
		return "", nil, err
	}

	jwtToken, err := jwt.Issue(user.ID, s.jwtSecret, s.jwtTTLHours)
	return jwtToken, user, err
}
