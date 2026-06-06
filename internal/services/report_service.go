package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rize-api/internal/models"
	"rize-api/internal/repository"
)

type ReportService struct {
	reportRepo *repository.ReportRepository
	sitRepo    *repository.SitRepository
	moveRepo   *repository.MoveRepository
	userRepo   *repository.UserRepository
}

func NewReportService(
	reportRepo *repository.ReportRepository,
	sitRepo *repository.SitRepository,
	moveRepo *repository.MoveRepository,
	userRepo *repository.UserRepository,
) *ReportService {
	return &ReportService{
		reportRepo: reportRepo,
		sitRepo:    sitRepo,
		moveRepo:   moveRepo,
		userRepo:   userRepo,
	}
}

func (s *ReportService) GetToday(ctx context.Context, userID string) (*models.DailyReport, error) {
	today := time.Now().Format("2006-01-02")
	return s.reportRepo.GetByDate(ctx, userID, today)
}

func (s *ReportService) GetHistory(ctx context.Context, userID string, days int) ([]*models.DailyReport, error) {
	if days <= 0 || days > 90 {
		days = 7
	}
	return s.reportRepo.GetHistory(ctx, userID, days)
}

func (s *ReportService) Generate(ctx context.Context, userID string) (*models.DailyReport, error) {
	today := time.Now().Format("2006-01-02")
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return s.buildAndSave(ctx, user, today)
}

func (s *ReportService) GenerateAllDailyReports(ctx context.Context) {
	users, err := s.userRepo.GetAllWithPushEnabled(ctx)
	if err != nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	for _, u := range users {
		s.buildAndSave(ctx, u, today) //nolint:errcheck
	}
}

func (s *ReportService) GetUsersOverSitLimit(ctx context.Context) ([]string, error) {
	return s.sitRepo.GetUsersOverSitLimit(ctx)
}

func (s *ReportService) GetNoMovementUsers(ctx context.Context) ([]string, error) {
	return s.moveRepo.GetNoMovementUsers(ctx)
}

func (s *ReportService) buildAndSave(ctx context.Context, user *models.User, date string) (*models.DailyReport, error) {
	sits, err := s.sitRepo.GetByDate(ctx, user.ID, date)
	if err != nil {
		return nil, err
	}
	moves, err := s.moveRepo.GetByDate(ctx, user.ID, date)
	if err != nil {
		return nil, err
	}

	var totalSit, longestSit int
	for _, ss := range sits {
		totalSit += ss.DurationMin
		if ss.DurationMin > longestSit {
			longestSit = ss.DurationMin
		}
	}

	var totalMove int
	for _, ms := range moves {
		totalMove += ms.DurationMin
	}

	score := calculateHealthScore(totalSit, totalMove, longestSit, len(moves), user.SitLimitMin)
	advice := generateAdvice(totalMove, longestSit, len(sits), len(moves), score, user.SitLimitMin, user.MoveGoalMin)

	report := &models.DailyReport{
		UserID:            user.ID,
		ReportDate:        date,
		TotalSitMin:       totalSit,
		TotalMoveMin:      totalMove,
		LongestSitMin:     longestSit,
		SitSessionsCount:  len(sits),
		MoveSessionsCount: len(moves),
		HealthScore:       score,
		Advice:            &advice,
	}
	return s.reportRepo.Upsert(ctx, report)
}

func calculateHealthScore(totalSitMin, totalMoveMin, longestSitMin, moveSessions, sitLimit int) int {
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

func generateAdvice(totalMoveMin, longestSitMin, sitCount, moveCount, score, sitLimit, moveGoal int) string {
	var tips []string
	if longestSitMin > sitLimit {
		tips = append(tips, fmt.Sprintf(
			"Your longest sitting stretch was %d min — %d min over your limit. Tomorrow, set a phone alarm for %d min after each break.",
			longestSitMin, longestSitMin-sitLimit, sitLimit,
		))
	}
	if totalMoveMin < moveGoal {
		deficit := moveGoal - totalMoveMin
		tips = append(tips, fmt.Sprintf(
			"You moved %d min today, %d min short of your goal. Add a 10-minute walk after lunch tomorrow.",
			totalMoveMin, deficit,
		))
	}
	if sitCount > 0 && moveCount == 0 {
		tips = append(tips, "You had no movement sessions today. Even 5 minutes of standing counts — start there.")
	}
	if score >= 80 {
		tips = append(tips, "Great day! You stayed disciplined. Keep this rhythm going tomorrow.")
	}
	if len(tips) == 0 {
		return "Solid day. Stay consistent."
	}
	return strings.Join(tips, " ")
}
