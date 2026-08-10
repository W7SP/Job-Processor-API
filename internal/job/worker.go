package job

import (
	"context"
	"log"
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
	jobs  <-chan *Job   // read-only end of the channel — workers only consume
	store StatusUpdater // only needs to update status, not create/get
}

func NewWorkerPool(jobs <-chan *Job, store StatusUpdater) *WorkerPool {
	return &WorkerPool{jobs: jobs, store: store}
}

// LaunchRoutines launches n goroutines. Now they are waiting to start working the moment there is a job in the channel.
// The goroutines run in the background until ctx is canceled.
func (wp *WorkerPool) LaunchRoutines(n int) {
	for i := range n {
		go wp.runWorker(i)
	}
}

func (wp *WorkerPool) runWorker(id int) {
	for j := range wp.jobs {
		log.Printf("worker %d: picked up job %s (%s)", id, j.ID, j.Payload)
		time.Sleep(10 * time.Second) // pretend to do work
		err := wp.store.UpdateStatus(j.ID, "done")
		if err != nil {
			log.Printf("OOPSY DAISYY %v", err)
			return
		}

		log.Printf("worker %d: finished job %s", id, j.ID)
	}
}
