package internal

import (
	"context"
	"log/slog"
	"time"
)

// RunCleanup periodically removes old diagrams until ctx is cancelled.
func RunCleanup(ctx context.Context, logger *slog.Logger, generator Generator) {
	const (
		runEvery    = 5 * time.Minute
		cleanupLast = time.Hour
	)

	ticker := time.NewTicker(runEvery)
	defer ticker.Stop()

	for {
		if err := generator.CleanUp(cleanupLast); err != nil {
			logger.Error("cleanup failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
