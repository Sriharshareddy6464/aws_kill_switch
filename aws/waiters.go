package aws

import (
	"context"
	"fmt"
	"time"
)

// PollResource repeatedly checks condition until deleted or timeout
func PollResource(ctx context.Context, checkFunc func() (bool, error), interval, timeout time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timeout waiting for resource status change")
		case <-ticker.C:
			deleted, err := checkFunc()
			if err != nil {
				return err
			}
			if deleted {
				return nil
			}
		}
	}
}
