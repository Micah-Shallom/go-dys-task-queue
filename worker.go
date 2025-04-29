package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Worker struct {
	id      int
	queue   *PriorityQueue
	metrics *Metrics
	stop    chan struct{}
}

func (w *Worker) Stop() {
	close(w.stop)
}

func NewWorker(id int, queue *PriorityQueue, metrics *Metrics) *Worker {
	return &Worker{
		id:      id,
		queue:   queue,
		metrics: metrics,
		stop:    make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		return
	case <-w.stop:
		return
	default:
		task := w.queue.Dequeue()
		if err := w.processTask(task); err != nil {
			w.metrics.RecordFailure()
			fmt.Printf("Worker %d: Failed task %d: %v\n", w.id, task.ID, err)
		} else {
			w.metrics.RecordSuccess()
		}
	}
}

func (w *Worker) processTask(task Task) error {
	fmt.Printf("Worker %d: Processing task %d (Priority: %d, Name: %s)\n", w.id, task.ID, task.Priority, task.Name)
	time.Sleep(2 * time.Second) //simulate work
	fmt.Printf("Worker %d: Completed task %d\n", w.id, task.ID)
	return nil
}
