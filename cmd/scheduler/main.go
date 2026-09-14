package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-api/cmd/scheduler/di"
	"go-api/internal/infrastructure/config"
	"go-api/internal/infrastructure/persistence/schema"
)

func main() {
	env := config.Load()
	db := config.ConnectDatabase(env)

	if err := schema.AssertModelsMatchDB(db); err != nil {
		log.Fatalf("schema check failed: %v", err)
	}

	container := di.NewContainer(db, env)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	runStaleMediaPoller(ctx, container, env.SchedulerInterval)
}

func runStaleMediaPoller(ctx context.Context, container *di.Container, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	log.Printf(
		"scheduler: stale processing media poller started (interval=%s staleAfter=%s maxAge=%s clock-aligned)",
		interval,
		container.StaleAfter,
		container.MaxAge,
	)

	tickStaleMedias(ctx, container)

	for {
		wait := durationUntilNextInterval(time.Now(), interval)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Println("scheduler: stale processing media poller stopped")
			return
		case <-timer.C:
			tickStaleMedias(ctx, container)
		}
	}
}

func tickStaleMedias(ctx context.Context, container *di.Container) {
	startedAt := time.Now().UTC()
	result, err := container.RecoverStaleProcessingMediasHandler.Handle(ctx, startedAt)
	if err != nil {
		log.Printf("scheduler: recover stale processing medias failed: %v", err)
		return
	}
	log.Printf(
		"scheduler: stale medias scanned=%d requeued=%d failed=%d errors=%d duration=%s",
		result.Scanned,
		result.Requeued,
		result.Failed,
		result.Errors,
		time.Since(startedAt).Round(time.Millisecond),
	)
}
