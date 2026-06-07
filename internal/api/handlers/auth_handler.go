package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"rize-api/internal/api/middleware"
	"rize-api/internal/services"
	"rize-api/pkg/response"
	"rize-api/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc         *services.AuthService
	jwtTTLHours int
	frontendURL string
}

func NewAuthHandler(svc *services.AuthService, jwtTTLHours int, frontendURL string) *AuthHandler {
	return &AuthHandler{svc: svc, jwtTTLHours: jwtTTLHours, frontendURL: frontendURL}
}

// --- Magic link ---

type magicInput struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *AuthHandler) SendMagicLink(c *fiber.Ctx) error {
	var in magicInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "invalid email")
	}
	if err := h.svc.SendMagicLink(c.Context(), in.Email); err != nil {
		return response.InternalError(c)
	}
	return response.Message(c, "magic link sent — check your email")
}

func (h *AuthHandler) VerifyMagicLink(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return response.BadRequest(c, "missing token")
	}
	jwtToken, _, err := h.svc.VerifyMagicLink(c.Context(), token)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	h.setAuthCookie(c, jwtToken)
	return c.Redirect(h.frontendURL, fiber.StatusFound)
}

// --- Email + password ---

type registerInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var in registerInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "invalid input")
	}
	if err := h.svc.Register(c.Context(), in.Email, in.Password); err != nil {
		if err.Error() == "email already registered" {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c)
	}
	return response.Message(c, "verification email sent — check your inbox")
}

type loginInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var in loginInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "invalid input")
	}
	jwtToken, _, err := h.svc.Login(c.Context(), in.Email, in.Password)
	if err != nil {
		if err.Error() == "email not verified — check your inbox" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}
	h.setAuthCookie(c, jwtToken)
	return response.Message(c, "logged in")
}

func (h *AuthHandler) ResendVerification(c *fiber.Ctx) error {
	var in struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "invalid email")
	}
	_ = h.svc.ResendVerification(c.Context(), in.Email) // always silent
	return response.Message(c, "if that email needs verification, we've sent a new link")
}

func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return response.BadRequest(c, "missing token")
	}
	if err := h.svc.VerifyEmail(c.Context(), token); err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.Message(c, "email verified")
}

// --- Google OAuth ---

// GoogleExchange is called by the frontend callback page with the authorization code.
func (h *AuthHandler) GoogleExchange(c *fiber.Ctx) error {
	var in struct {
		Code string `json:"code" validate:"required"`
	}
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "missing code")
	}
	jwtToken, _, err := h.svc.HandleGoogleCallback(c.Context(), in.Code)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "google auth failed"})
	}
	h.setAuthCookie(c, jwtToken)
	return response.Message(c, "logged in")
}

func (h *AuthHandler) GoogleRedirect(c *fiber.Ctx) error {
	state, err := randomHex(16)
	if err != nil {
		return response.InternalError(c)
	}
	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		MaxAge:   300,
		Path:     "/",
	})
	return c.Redirect(h.svc.GoogleAuthURL(state), fiber.StatusFound)
}

func (h *AuthHandler) GoogleCallback(c *fiber.Ctx) error {
	state := c.Query("state")
	if state == "" || state != c.Cookies("oauth_state") {
		return c.Redirect(h.frontendURL+"/login?error=invalid_state", fiber.StatusFound)
	}
	c.Cookie(&fiber.Cookie{Name: "oauth_state", Value: "", Expires: time.Unix(0, 0), Path: "/"})

	code := c.Query("code")
	if code == "" {
		return c.Redirect(h.frontendURL+"/login?error=no_code", fiber.StatusFound)
	}

	jwtToken, _, err := h.svc.HandleGoogleCallback(c.Context(), code)
	if err != nil {
		return c.Redirect(h.frontendURL+"/login?error=oauth_failed", fiber.StatusFound)
	}
	h.setAuthCookie(c, jwtToken)
	return c.Redirect(h.frontendURL, fiber.StatusFound)
}

// --- Shared ---

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "rize_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None",
		Expires:  time.Unix(0, 0),
		Path:     "/",
	})
	return response.Message(c, "logged out")
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	user, err := h.svc.GetUser(c.Context(), userID)
	if err != nil || user == nil {
		return response.Unauthorized(c)
	}
	return response.OK(c, user)
}

func (h *AuthHandler) setAuthCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "rize_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "None", // cross-domain: frontend (vercel) → backend (railway)
		Expires:  time.Now().Add(time.Duration(h.jwtTTLHours) * time.Hour),
		Path:     "/",
	})
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
