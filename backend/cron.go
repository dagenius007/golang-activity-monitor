package main

import (
	"context"

	"binalyze-test/setup"

	"github.com/robfig/cron/v3"
)

func RunSchedule(services *setup.ServiceDependencies) {
	ctx := context.Background()
	s := cron.New()

	s.AddFunc("@every 10s", func() {
		services.Logger.Info("Running cron service")
		services.ProcessService.FetchAndInsertProcess(ctx)
	})

	s.Start()
}
