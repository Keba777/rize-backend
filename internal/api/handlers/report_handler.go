package handlers

import (
	"strconv"

	"rize-api/internal/api/middleware"
	"rize-api/internal/models"
	"rize-api/internal/services"
	"rize-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type ReportHandler struct {
	svc *services.ReportService
}

func NewReportHandler(svc *services.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) Today(c *fiber.Ctx) error {
	report, err := h.svc.GetToday(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	return response.OK(c, report)
}

func (h *ReportHandler) History(c *fiber.Ctx) error {
	days, _ := strconv.Atoi(c.Query("days", "7"))
	reports, err := h.svc.GetHistory(c.Context(), middleware.GetUserID(c), days)
	if err != nil {
		return response.InternalError(c)
	}
	if reports == nil {
		reports = []*models.DailyReport{}
	}
	return response.OK(c, reports)
}

func (h *ReportHandler) Generate(c *fiber.Ctx) error {
	report, err := h.svc.Generate(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OK(c, report)
}
