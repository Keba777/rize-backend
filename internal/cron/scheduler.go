package cron

import (
	"context"
	"log"
	"time"

	"rize-api/internal/services"

	"github.com/robfig/cron/v3"
)

func Setup(reportSvc *services.ReportService, pushSvc *services.PushService) {
	c := cron.New()

	// Every 5 min: notify users sitting over their limit
	c.AddFunc("*/5 * * * *", func() { //nolint:errcheck
		ctx := context.Background()
		ids, err := reportSvc.GetUsersOverSitLimit(ctx)
		if err != nil {
			log.Printf("cron sit check: %v", err)
			return
		}
		for _, id := range ids {
			// rough elapsed: query active session directly would be better,
			// but we use sit_limit_min + 5 as a safe over-estimate
			pushSvc.SendSitReminder(ctx, id, 0)
		}
	})

	// 9 PM UTC: generate daily reports and send push
	c.AddFunc("0 21 * * *", func() { //nolint:errcheck
		ctx := context.Background()
		reportSvc.GenerateAllDailyReports(ctx)
		pushSvc.SendDailyReportNotifications(ctx)
	})

	// Noon: nudge users with no movement logged yet
	c.AddFunc("0 12 * * *", func() { //nolint:errcheck
		ctx := context.Background()
		ids, err := reportSvc.GetNoMovementUsers(ctx)
		if err != nil {
			return
		}
		for _, id := range ids {
			pushSvc.SendNoonNudgeToUser(ctx, id)
		}
	})

	c.Start()
	log.Printf("cron scheduler started at %s", time.Now().Format(time.RFC3339))
}
