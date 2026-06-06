package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"rize-api/internal/models"
	"rize-api/internal/repository"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type PushService struct {
	pushRepo    *repository.PushRepository
	userRepo    *repository.UserRepository
	reportRepo  *repository.ReportRepository
	sitRepo     *repository.SitRepository
	vapidPub    string
	vapidPriv   string
	vapidEmail  string
}

func NewPushService(
	pushRepo *repository.PushRepository,
	userRepo *repository.UserRepository,
	reportRepo *repository.ReportRepository,
	sitRepo *repository.SitRepository,
	vapidPub, vapidPriv, vapidEmail string,
) *PushService {
	return &PushService{
		pushRepo:   pushRepo,
		userRepo:   userRepo,
		reportRepo: reportRepo,
		sitRepo:    sitRepo,
		vapidPub:   vapidPub,
		vapidPriv:  vapidPriv,
		vapidEmail: vapidEmail,
	}
}

func (s *PushService) Save(ctx context.Context, userID string, in *models.SavePushInput) error {
	if err := s.pushRepo.Save(ctx, userID, in); err != nil {
		return err
	}
	return s.userRepo.SetPushEnabled(ctx, userID, true)
}

func (s *PushService) Delete(ctx context.Context, userID, endpoint string) error {
	return s.pushRepo.Delete(ctx, userID, endpoint)
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon"`
	Data  struct {
		URL string `json:"url"`
	} `json:"data"`
}

func (s *PushService) SendToUser(ctx context.Context, userID string, p pushPayload) {
	subs, err := s.pushRepo.GetByUserID(ctx, userID)
	if err != nil || len(subs) == 0 {
		return
	}
	payload, _ := json.Marshal(p)
	for _, sub := range subs {
		s.send(sub, payload)
	}
}

func (s *PushService) send(sub *models.PushSubscription, payload []byte) {
	_, err := webpush.SendNotification(payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}, &webpush.Options{
		VAPIDPublicKey:  s.vapidPub,
		VAPIDPrivateKey: s.vapidPriv,
		Subscriber:      s.vapidEmail,
		TTL:             30,
	})
	if err != nil {
		log.Printf("push send error for endpoint %s: %v", sub.Endpoint, err)
	}
}

func (s *PushService) SendSitReminder(ctx context.Context, userID string, elapsedMin int) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return
	}
	name := "there"
	if user.Name != nil {
		name = *user.Name
	}
	hours := elapsedMin / 60
	mins := elapsedMin % 60
	var elapsed string
	if hours > 0 {
		elapsed = fmt.Sprintf("%dh %dm", hours, mins)
	} else {
		elapsed = fmt.Sprintf("%dm", mins)
	}
	p := pushPayload{
		Title: fmt.Sprintf("Time to move, %s.", name),
		Body:  fmt.Sprintf("You've been sitting for %s. Stand up for 5 minutes.", elapsed),
		Icon:  "/icons/icon-192.png",
	}
	p.Data.URL = "/move"
	s.SendToUser(ctx, userID, p)
}

func (s *PushService) SendDailyReportNotifications(ctx context.Context) {
	users, err := s.userRepo.GetAllWithPushEnabled(ctx)
	if err != nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	for _, u := range users {
		report, err := s.reportRepo.GetByDate(ctx, u.ID, today)
		if err != nil || report == nil {
			continue
		}
		name := "there"
		if u.Name != nil {
			name = *u.Name
		}
		p := pushPayload{
			Title: "Today's Rize Report",
			Body: fmt.Sprintf("Score: %d/100 · Moved %dmin · Longest sit: %dmin, %s",
				report.HealthScore, report.TotalMoveMin, report.LongestSitMin, name),
			Icon: "/icons/icon-192.png",
		}
		p.Data.URL = "/report"
		s.SendToUser(ctx, u.ID, p)
	}
}

func (s *PushService) GetSubsByUserID(ctx context.Context, userID string) ([]*models.PushSubscription, error) {
	return s.pushRepo.GetByUserID(ctx, userID)
}

func (s *PushService) SendNoonNudgeToUser(ctx context.Context, userID string) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return
	}
	p := pushPayload{
		Title: "Haven't moved yet today.",
		Body:  "A 10-minute walk now puts you on track. You got this.",
		Icon:  "/icons/icon-192.png",
	}
	p.Data.URL = "/move"
	s.SendToUser(ctx, userID, p)
}

func (s *PushService) GetVAPIDPublicKey() string {
	return s.vapidPub
}
