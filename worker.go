package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Worker struct {
	id         int
	JobChannel chan Job
	metrics    *Metrics
	running    bool
}

func NewWorker(id int, metrics *Metrics) *Worker {
	return &Worker{
		id:         id,
		JobChannel: make(chan Job),
		metrics:    metrics,
		running:    false,
	}
}

func (w *Worker) Start(ctx context.Context, stopCh <-chan struct{}, wg *sync.WaitGroup, queue *PriorityJobQueue) {
	defer wg.Done()
	w.running = true

	fmt.Printf("👷 Worker %d: Started and ready to process tasks\n", w.id)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("👷 Worker %d: Context canceled, stopping\n", w.id)
			return
		case <-stopCh:
			fmt.Printf("👷 Worker %d: Received stop signal, stopping\n", w.id)
			return
		case job := <- w.JobChannel:
			fmt.Printf("👷 Worker %d: Received job from channel\n", w.id)
			w.metrics.IncrementActiveWorkers()
			err := w.processTask(job, time.Now())
			if err != nil {
				w.metrics.RecordFailure()
				fmt.Printf("🔴 Worker %d: Failed to process task %d\n", w.id, job.task.ID)
			}
			w.metrics.DecrementActiveWorkers()
		}
	}
}

func (w *Worker) processTask(job Job, startTime time.Time) error {
	w.running = true
	defer func() {
		w.running = false
	}()

	fmt.Printf("Worker %d: Processing task %d (Priority: %d, Name: %s)\n", w.id, job.task.ID, job.task.Priority, job.task.Name)
	time.Sleep(2 * time.Second) //simulate work

	// Simulate failure for ~10% of tasks
	if rand.Float32() < 0.1 {
		return fmt.Errorf("simulated failure for task %d", job.task.ID)
	}

	if job.task.Priority == 1 {
		w.metrics.RecordSuccess()
		fmt.Printf("🔴 Worker %d: Completed HIGH priority task %d in %v\n",
			w.id, job.task.ID, time.Since(startTime))
	} else if job.task.Priority == 2 {
		// Medium priority tasks
		w.metrics.RecordSuccess()
		fmt.Printf("🟠 Worker %d: Completed MEDIUM priority task %d in %v\n",
			w.id, job.task.ID, time.Since(startTime))
	} else {
		// Low priority tasks
		w.metrics.RecordSuccess()
		fmt.Printf("🟢 Worker %d: Completed LOW priority task %d in %v\n",
			w.id, job.task.ID, time.Since(startTime))
	}
	return nil
}
