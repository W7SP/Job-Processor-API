package bike

import (
	"context"
	"log"
	"sync"
	"time"
)

type WorkerPool struct {
	bikeChan chan *Bike
	wg       sync.WaitGroup
	store    StatusUpdater
}

func NewWorkerPool(bikeChan chan *Bike, store StatusUpdater) *WorkerPool {
	return &WorkerPool{bikeChan: bikeChan, store: store}
}

func (wp *WorkerPool) LaunchRoutines(n int) {
	for i := range n {
		wp.wg.Add(1)
		go wp.Work(i)
	}
}

func (wp *WorkerPool) Work(id int) {
	defer wp.wg.Done()
	for bike := range wp.bikeChan {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := wp.store.UpdateStatus(ctx, bike.ID, "assembled")
		cancel()
		if err != nil { // pass context here
			log.Printf("worker %d: failed to update status of bike ID: %s: %v", id, bike.ID, err)
			continue
		}
	}
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
