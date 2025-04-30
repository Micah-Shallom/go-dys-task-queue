package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Worker struct {
	id         int
	queue      *PriorityJobQueue
	JobChannel chan Job
	metrics    *Metrics
	running    bool
}

func NewWorker(id int, queue *PriorityJobQueue, metrics *Metrics) *Worker {
	return &Worker{
		id:         id,
		queue:      queue,
		JobChannel: make(chan Job),
		metrics:    metrics,
		running:    false,
	}
}


func (w *Worker) Start(ctx context.Context, stopCh <-chan struct{}, wg *sync.WaitGroup) {
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
		default:
			// continue processing
		}

		select {
		case job := <- w.queue.jobqueue:
			startTime := time.Now()
			w.processTask(job, startTime)

		case <-time.After(100 * time.Millisecond):
			select {
			case <-ctx.Done():
				fmt.Printf("👷 Worker %d: Context canceled during wait, stopping\n", w.id)
				return
			case <-stopCh:
				fmt.Printf("👷 Worker %d: Received stop signal during wait, stopping\n", w.id)
				return
			default:
				// Continue the loop and try to get another task
			}
		}
	}
}

func (w *Worker) processTask(job Job, startTime time.Time) error {
	fmt.Printf("Worker %d: Processing task %d (Priority: %d, Name: %s)\n", w.id, job.task.ID, job.task.Priority, job.task.Name)
	time.Sleep(2 * time.Second) //simulate work

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
