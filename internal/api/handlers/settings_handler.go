package handlers

import (
	"rize-api/internal/api/middleware"
	"rize-api/internal/models"
	"rize-api/internal/repository"
	"rize-api/internal/services"
	"rize-api/pkg/response"
	"rize-api/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

type SettingsHandler struct {
	userRepo *repository.UserRepository
	pushSvc  *services.PushService
}

func NewSettingsHandler(userRepo *repository.UserRepository, pushSvc *services.PushService) *SettingsHandler {
	return &SettingsHandler{userRepo: userRepo, pushSvc: pushSvc}
}

func (h *SettingsHandler) Update(c *fiber.Ctx) error {
	var in models.UpdateSettingsInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "validation failed")
	}
	user, err := h.userRepo.UpdateSettings(c.Context(), middleware.GetUserID(c), &in)
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, user)
}

func (h *SettingsHandler) SavePush(c *fiber.Ctx) error {
	var in models.SavePushInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "validation failed")
	}
	if err := h.pushSvc.Save(c.Context(), middleware.GetUserID(c), &in); err != nil {
		return response.InternalError(c)
	}
	return response.Message(c, "push subscription saved")
}

func (h *SettingsHandler) DeletePush(c *fiber.Ctx) error {
	var in models.DeletePushInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if err := h.pushSvc.Delete(c.Context(), middleware.GetUserID(c), in.Endpoint); err != nil {
		return response.InternalError(c)
	}
	return response.NoContent(c)
}

func (h *SettingsHandler) GetVAPIDKey(c *fiber.Ctx) error {
	return response.OK(c, fiber.Map{"public_key": h.pushSvc.GetVAPIDPublicKey()})
}
