package handlers

import (
	"time"

	"rize-api/internal/api/middleware"
	"rize-api/internal/models"
	"rize-api/internal/repository"
	"rize-api/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	sitRepo    *repository.SitRepository
	moveRepo   *repository.MoveRepository
	reportRepo *repository.ReportRepository
	userRepo   *repository.UserRepository
}

func NewDashboardHandler(
	sitRepo *repository.SitRepository,
	moveRepo *repository.MoveRepository,
	reportRepo *repository.ReportRepository,
	userRepo *repository.UserRepository,
) *DashboardHandler {
	return &DashboardHandler{
		sitRepo:    sitRepo,
		moveRepo:   moveRepo,
		reportRepo: reportRepo,
		userRepo:   userRepo,
	}
}

type TodayStats struct {
	ActiveSit    *models.SitSession   `json:"active_sit"`
	TodaySit     []*models.SitSession `json:"today_sit"`
	TodayMove    []*models.MoveSession `json:"today_move"`
	TotalSitMin  int                  `json:"total_sit_min"`
	TotalMoveMin int                  `json:"total_move_min"`
	SitLimitMin  int                  `json:"sit_limit_min"`
	MoveGoalMin  int                  `json:"move_goal_min"`
}

type ScoreBreakdown struct {
	Score           int    `json:"score"`
	TotalSitMin     int    `json:"total_sit_min"`
	TotalMoveMin    int    `json:"total_move_min"`
	LongestSitMin   int    `json:"longest_sit_min"`
	MoveSessions    int    `json:"move_sessions"`
	SitLimitMin     int    `json:"sit_limit_min"`
	Label           string `json:"label"`
}

func (h *DashboardHandler) Today(c *fiber.Ctx) error {
	ctx := c.Context()
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return response.Unauthorized(c)
	}

	activeSit, _ := h.sitRepo.GetActive(ctx, userID)
	todaySit, _ := h.sitRepo.GetToday(ctx, userID)
	todayMove, _ := h.moveRepo.GetToday(ctx, userID)

	if todaySit == nil {
		todaySit = []*models.SitSession{}
	}
	if todayMove == nil {
		todayMove = []*models.MoveSession{}
	}

	var totalSit, totalMove int
	for _, s := range todaySit {
		totalSit += s.DurationMin
	}
	for _, m := range todayMove {
		totalMove += m.DurationMin
	}

	return response.OK(c, TodayStats{
		ActiveSit:    activeSit,
		TodaySit:     todaySit,
		TodayMove:    todayMove,
		TotalSitMin:  totalSit,
		TotalMoveMin: totalMove,
		SitLimitMin:  user.SitLimitMin,
		MoveGoalMin:  user.MoveGoalMin,
	})
}

func (h *DashboardHandler) Score(c *fiber.Ctx) error {
	ctx := c.Context()
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return response.Unauthorized(c)
	}

	today := time.Now().Format("2006-01-02")
	todaySit, _ := h.sitRepo.GetByDate(ctx, userID, today)
	todayMove, _ := h.moveRepo.GetByDate(ctx, userID, today)

	var totalSit, longestSit, totalMove int
	for _, s := range todaySit {
		totalSit += s.DurationMin
		if s.DurationMin > longestSit {
			longestSit = s.DurationMin
		}
	}
	for _, m := range todayMove {
		totalMove += m.DurationMin
	}

	score := calcScore(totalSit, totalMove, longestSit, len(todayMove), user.SitLimitMin)
	label := scoreLabel(score)

	return response.OK(c, ScoreBreakdown{
		Score:         score,
		TotalSitMin:   totalSit,
		TotalMoveMin:  totalMove,
		LongestSitMin: longestSit,
		MoveSessions:  len(todayMove),
		SitLimitMin:   user.SitLimitMin,
		Label:         label,
	})
}

func calcScore(totalSitMin, totalMoveMin, longestSitMin, moveSessions, sitLimit int) int {
	score := 100
	if longestSitMin > sitLimit {
		score -= min((longestSitMin-sitLimit)*2, 40)
	}
	if totalSitMin > 360 {
		score -= min((totalSitMin-360)/10, 20)
	}
	score += min(moveSessions*5, 20)
	if totalMoveMin >= 30 {
		score += 10
	}
	return max(0, min(100, score))
}

func scoreLabel(score int) string {
	switch {
	case score >= 90:
		return "Excellent day"
	case score >= 75:
		return "Good day"
	case score >= 60:
		return "Doing okay"
	case score >= 40:
		return "Needs improvement"
	default:
		return "Rough day"
	}
}
