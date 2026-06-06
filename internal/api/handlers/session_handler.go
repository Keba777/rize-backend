package handlers

import (
	"strconv"

	"rize-api/internal/api/middleware"
	"rize-api/internal/models"
	"rize-api/internal/services"
	"rize-api/pkg/response"
	"rize-api/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

type SessionHandler struct {
	svc *services.SessionService
}

func NewSessionHandler(svc *services.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

// Sit

func (h *SessionHandler) StartSit(c *fiber.Ctx) error {
	s, err := h.svc.StartSit(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	return response.Created(c, s)
}

func (h *SessionHandler) EndSit(c *fiber.Ctx) error {
	s, err := h.svc.EndSit(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	if s == nil {
		return response.NotFound(c, "no active sit session")
	}
	return response.OK(c, s)
}

func (h *SessionHandler) GetActiveSit(c *fiber.Ctx) error {
	s, err := h.svc.GetActiveSit(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, s) // null is valid — no active session
}

func (h *SessionHandler) GetTodaySit(c *fiber.Ctx) error {
	sessions, err := h.svc.GetTodaySit(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	if sessions == nil {
		sessions = []*models.SitSession{}
	}
	return response.OK(c, sessions)
}

// Move

func (h *SessionHandler) LogMove(c *fiber.Ctx) error {
	var in models.LogMoveInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "validation failed")
	}
	s, err := h.svc.LogMove(c.Context(), middleware.GetUserID(c), &in)
	if err != nil {
		return response.InternalError(c)
	}
	return response.Created(c, s)
}

func (h *SessionHandler) StartMove(c *fiber.Ctx) error {
	var in models.StartMoveInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if errs := validator.Validate(in); errs != nil {
		return response.BadRequest(c, "validation failed")
	}
	s, err := h.svc.StartMove(c.Context(), middleware.GetUserID(c), &in)
	if err != nil {
		return response.InternalError(c)
	}
	return response.Created(c, s)
}

func (h *SessionHandler) EndMove(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "invalid session id")
	}
	s, err := h.svc.EndMove(c.Context(), middleware.GetUserID(c), id)
	if err != nil {
		return response.InternalError(c)
	}
	if s == nil {
		return response.NotFound(c, "session not found")
	}
	return response.OK(c, s)
}

func (h *SessionHandler) GetTodayMove(c *fiber.Ctx) error {
	sessions, err := h.svc.GetTodayMove(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	if sessions == nil {
		sessions = []*models.MoveSession{}
	}
	return response.OK(c, sessions)
}

func (h *SessionHandler) DeleteMove(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "invalid session id")
	}
	deleted, err := h.svc.DeleteMove(c.Context(), middleware.GetUserID(c), id)
	if err != nil {
		return response.InternalError(c)
	}
	if !deleted {
		return response.NotFound(c, "session not found")
	}
	return response.NoContent(c)
}
