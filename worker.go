package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

type Worker struct {
	id              int
	JobChannel      chan Job
	maxJobPerWorker int32
	metrics         *Metrics
	stopWorkerChan  chan struct{} //channel to signal worker is busy
	jobCount        int32         //tracks the number of jobs in the worker job channel
	idleSince       atomic.Value
	idleTimeout     time.Duration
}

func NewWorker(id int, metrics *Metrics, maxJobPerWorker int32) *Worker {
	return &Worker{
		id:              id,
		maxJobPerWorker: maxJobPerWorker,
		JobChannel:      make(chan Job, maxJobPerWorker),
		metrics:         metrics,
		stopWorkerChan:  make(chan struct{}),
		jobCount:        0,
		idleTimeout:     10 * time.Second, // Set idle timeout to 5 seconds
	}
}

func (w *Worker) Stop() {
	fmt.Printf("👷 Worker %d: Stopping\n", w.id)
	close(w.JobChannel)
	w.metrics.DecrementActiveWorkers()
}

func (w *Worker) signalAvailability(d *Dispatcher) {
	timeout := 5 * time.Second // Set a timeout for finding an available worker

	if w.GetJobCount() < w.maxJobPerWorker {
		select {
		case d.availableWorkers <- w:
			//successfully signaled as available
		case <-time.After(timeout):
			//timeout, worker is busy
			fmt.Printf("👷 Worker %d: No available workers, busy processing tasks\n", w.id)
			return
		}
	}
}

func (w *Worker) Start(ctx context.Context, d *Dispatcher) {
	defer d.wg.Done()
	fmt.Printf("👷 Worker %d: Started and ready to process tasks\n", w.id)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	w.signalAvailability(d)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("👷 Worker %d: Context canceled, stopping\n", w.id)
			return
		case <-d.stopCh: //general signal from dispatcher
			fmt.Printf("👷 Worker %d: Received stop signal, stopping\n", w.id)
			w.metrics.DecrementActiveWorkers()
			return
		case <-w.stopWorkerChan: //signal to terminate worker
			fmt.Printf("👷 Worker %d: Received stop worker signal, stopping\n", w.id)
			w.metrics.DecrementActiveWorkers()
			return

		//remember to implement a centralized idle ticker system for higher load scenerios
		case <-ticker.C:
			w.signalAvailability(d)

			if w.idleTimeout > 0 && w.GetJobCount() == 0 && w.IsIdleLongEnough() {
				fmt.Printf("👷 Worker %d: Idle for too long, stopping\n", w.id)
				w.Stop()
				return
			}

		case job := <-w.JobChannel:
			err := w.processTask(job, time.Now())
			if err != nil {
				w.metrics.RecordFailure()
				fmt.Printf("🔴 Worker %d: Failed to process task %d\n", w.id, job.task.ID)
			}
			w.DecrementJobCount()
			w.signalAvailability(d)
			w.metrics.DecrementJobsQueueCount()
		}
	}
}

func (w *Worker) processTask(job Job, startTime time.Time) error {

	fmt.Printf("Worker %d: Processing task %d (Priority: %d, Name: %s)\n", w.id, job.task.ID, job.task.Priority, job.task.Name)
	time.Sleep(2 * time.Second) //simulate work

	// Simulate failure for ~20% of tasks
	if rand.Float32() < 0.2 {
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

func (w *Worker) IncrementJobCount() {
	atomic.AddInt32((*int32)(&w.jobCount), 1)

	w.idleSince.Store(time.Time{}) // Reset idle time
}

func (w *Worker) DecrementJobCount() {
	count := atomic.AddInt32((*int32)(&w.jobCount), -1)
	if count == 0 {
		w.idleSince.Store(time.Now())
	}
}

func (w *Worker) GetJobCount() int32 {
	return atomic.LoadInt32((*int32)(&w.jobCount))
}

func (w *Worker) IsIdleLongEnough() bool {
	idleTime, ok := w.idleSince.Load().(time.Time)
	if !ok || idleTime.IsZero() {
		return false
	}

	return time.Since(idleTime) >= w.idleTimeout
}
