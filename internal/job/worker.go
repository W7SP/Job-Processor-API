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
	jobs  <-chan *Job   // read-only end of the channel — workers only consume
	store StatusUpdater // only needs to update status, not create/get
	wg    sync.WaitGroup
}

func NewWorkerPool(jobs <-chan *Job, store StatusUpdater) *WorkerPool {
	return &WorkerPool{jobs: jobs, store: store}
}

// LaunchRoutines launches n goroutines. Now they are waiting to start working the moment there is a job in the channel.
// Workers exit after the jobs channel is closed and its queued jobs are processed.
func (wp *WorkerPool) LaunchRoutines(n int) {
	for i := range n {
		wp.wg.Add(1)
		go wp.runWorker(i)
	}
}

func (wp *WorkerPool) runWorker(id int) {
	defer wp.wg.Done()
	for j := range wp.jobs {
		log.Printf("worker %d: picked up job %s (%s)", id, j.ID, j.Payload)
		time.Sleep(10 * time.Second) // pretend to do work
		err := wp.store.UpdateStatus(j.ID, "done")
		if err != nil {
			log.Printf("worker %d: failed to update job %s: %v", id, j.ID, err)
			continue
		}

		log.Printf("worker %d: finished job %s", id, j.ID)
	}
}

// Wait blocks until all launched workers have exited. Close the jobs channel first.
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
