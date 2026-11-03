package job

import (
	"context"
	"log"
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
		log.Printf("worker %d: picked up job %s (%s)", id, j.ID, j.Payload)
		// Pick a job from the channel and change its status to running
		if err := wp.store.UpdateStatus(j.ID, "running"); err != nil {
			log.Printf("worker %d: failed to start job %s: %v", id, j.ID, err)
			continue
		}

		// Each job gets its own timeout, starting when processing begins.
		// Create a context with timeout after 10 seconds
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		// Attempt to update the status (nil=done / error or expired context=failed)
		err := wp.processor.Process(ctx, j)
		cancel() // Release the timer now, rather than when this worker exits.

		status := "done"
		if err != nil { // check if the process returned an error (not nil)
			status = "failed"
			log.Printf("worker %d: job %s failed: %v", id, j.ID, err)
		}

		err = wp.store.UpdateStatus(j.ID, status)
		if err != nil {
			log.Printf("worker %d: failed to update job %s: %v", id, j.ID, err)
			continue
		}

		log.Printf("worker %d: finished job %s with status %s", id, j.ID, status)
	}
}

// Wait blocks until all launched workers have exited. Close the jobs channel first.
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
