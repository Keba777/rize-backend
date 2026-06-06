package handlers

import (
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
	c.Cookie(&fiber.Cookie{
		Name:     "rize_token",
		Value:    jwtToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		Expires:  time.Now().Add(time.Duration(h.jwtTTLHours) * time.Hour),
		Path:     "/",
	})
	return c.Redirect(h.frontendURL, fiber.StatusFound)
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "rize_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
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
