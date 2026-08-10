package job

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
)

// SimulatedProcessor pretends to do work.
// Sleeps for a random duration, occasionally fails.
// This is our stand-in until there's real work to do.
type SimulatedProcessor struct{}

func (p *SimulatedProcessor) Process(ctx context.Context, j *Job) error {
	duration := time.Duration(1+rand.Intn(4)) * time.Second

	log.Printf("job %s: simulating %v of work", j.ID, duration)

	select {
	case <-time.After(duration):
		// Work finished in time.
		if rand.Intn(5) == 0 { // 20% random failure rate
			return fmt.Errorf("simulated failure for job %s", j.ID)
		}
		return nil

	case <-ctx.Done():
		// Context cancelled or timed out before work finished.
		return fmt.Errorf("job %s: context cancelled: %w", j.ID, ctx.Err())
	}
}
