package job

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Processor defines what it means to "process" a job.
// Right now we simulate work with a sleep, but this interface means
// later you could swap in a real image resize-er, email sender, etc.
// without touching the worker pool code at all.
type Processor interface {
	Process(ctx context.Context, j *Job) error
}

// WorkerPool manages a fixed number of goroutines that process jobs.
type WorkerPool struct {
	jobs      <-chan *Job   // read-only end of the channel — workers only consume
	store     StatusUpdater // only needs to update status, not create/get
	processor Processor
	wg        sync.WaitGroup
}

func NewWorkerPool(jobs <-chan *Job, store StatusUpdater, processor Processor) *WorkerPool {
	return &WorkerPool{jobs: jobs, store: store, processor: processor}
}

// LaunchRoutines launches n goroutines. Now they are waiting to start working the moment there is a job in the channel.
// Workers exit after the job channel is closed and its queued jobs are processed.
func (wp *WorkerPool) LaunchRoutines(n int) {
	for i := range n {
		wp.wg.Add(1)
		go wp.runWorker(i)
	}
}

func (wp *WorkerPool) runWorker(id int) {
	defer wp.wg.Done() // State that the worker finished (decrement wait group counter)
	for j := range wp.jobs {
		start := time.Now()
		// A logger that already carries these two fields, so every line below
		// gets "worker" and "job_id" without repeating them each time.
		logger := slog.With("worker", id, "job_id", j.ID)
		logger.Info("job picked up", "payload", j.Payload)

		// Pick a job from the channel and change its status to running
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := wp.store.UpdateStatus(ctx, j.ID, "running")
		cancel()
		if err != nil {
			logger.Error("failed to mark job running", "error", err)
			continue
		}

		// Each job gets its own timeout, starting when processing begins.
		// Create a context with timeout after 10 seconds
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		// Attempt to update the status (nil=done / error or expired context=failed)
		err = wp.processor.Process(ctx, j)
		cancel() // Release the timer now, rather than when this worker exits.

		status := "done"
		if err != nil { // check if the process returned an error (not nil)
			status = "failed"
			logger.Warn("job processing failed", "error", err)
		}
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		err = wp.store.UpdateStatus(ctx, j.ID, status)
		cancel()
		if err != nil {
			logger.Error("failed to save final status", "status", status, "error", err)
			continue
		}

		logger.Info("job finished", "status", status, "duration_ms", time.Since(start).Milliseconds())
	}
}

// Wait blocks until all launched workers have exited. Close the jobs channel first.
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
