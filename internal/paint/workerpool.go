package paint

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type WorkerPool struct {
	paintsChan chan *Paint
	wg         sync.WaitGroup
	store      StatusUpdater
}

func NewWorkerPool(paintsChan chan *Paint, store StatusUpdater) *WorkerPool {
	return &WorkerPool{paintsChan: paintsChan, store: store}
}

func (wp *WorkerPool) LaunchRoutines(n int) {
	for i := range n {
		wp.wg.Add(1)
		wp.Work(i)
	}
}

func (wp *WorkerPool) Work(id int) {
	defer wp.wg.Done()
	for paint := range wp.paintsChan {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := wp.store.UpdateStatus(ctx, paint.ID)
		cancel()
		if err != nil {
			fmt.Printf("worker %d: failed to update status of paint ID: %s: %v", id, paint.ID, err)
		}
	}
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
