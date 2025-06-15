package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync/atomic"
	"time"
)

type WorkerHandle struct {
	ID          int
	JobsChannel chan Job
}

type Worker struct {
	id                int
	JobChannel        chan Job
	maxJobPerWorker   int32
	metrics           *Metrics
	stopWorkerChan    chan struct{} //channel to signal worker is busy
	idleTerminationCh chan<- int    //channel to signal worker termination when idle
	jobCount          int32         //tracks the number of jobs in the worker job channel
	idleSince         atomic.Value
	idleTimeout       time.Duration
	config            TimeoutConfig
}

func NewWorker(id int, metrics *Metrics, maxJobPerWorker int32, d *Dispatcher) *Worker {
	return &Worker{
		id:                id,
		maxJobPerWorker:   maxJobPerWorker,
		JobChannel:        make(chan Job, maxJobPerWorker),
		metrics:           metrics,
		stopWorkerChan:    make(chan struct{}),
		idleTerminationCh: d.idleTerminationCh,
		jobCount:          0,
		idleTimeout:       DefaultTimeouts.WorkerIdleTimeout,
		config:            DefaultTimeouts,
	}
}

func (w *Worker) Stop(d *Dispatcher) {
	slog.Info("👷 Worker stopping", "worker_id", w.id)
	close(w.JobChannel)

	slog.Info("👷 Worker draining JobChannel and re-queueing tasks...", "worker_id", w.id)
	for job := range w.JobChannel {
		slog.Info("👷 Worker re-queueing task", "worker_id", w.id, "task_id", job.task.ID, "priority", job.task.Priority)
		if d == nil {
			slog.Error("👷 Worker failed to re-queue task", "worker_id", w.id, "task_id", job.task.ID, "error", "dispatcher is nil")
			continue
		}
		if err := d.queue.PushToHeap(job.task, nil); err != nil {
			slog.Error("👷 Worker failed to re-queue task", "worker_id", w.id, "task_id", job.task.ID, "error", err)
		}
	}

	slog.Info("👷 Worker finished draining JobChannel", "worker_id", w.id)
}

func (w *Worker) signalAvailability(d *Dispatcher) {
	d.setMU.Lock()
	defer d.setMU.Unlock()

	//only signal availability if worker is not already at max job capacity
	if w.GetJobCount() >= w.maxJobPerWorker {
		return
	}

	//only signal availability if worker is not already available
	if d.availableSet[w.id] {
		return
	}


	select {
	case d.availableWorkers <- w.id:
		//successfully signaled as available
		slog.Info("👷 Worker signaled availability", "worker_id", w.id, "current_jobs", w.GetJobCount())
		d.availableSet[w.id] = true // Mark worker as available
	default:
		// availableWorkers channel is full, do nothing
		slog.Debug("👷 Worker could not signal availability, channel full", "worker_id", w.id, "current_jobs", w.GetJobCount())
	}

}

func (w *Worker) Start(ctx context.Context, d *Dispatcher) {
	defer d.wg.Done()
	slog.Info("👷 Worker started and ready to process tasks", "worker_id", w.id)

	ticker := time.NewTicker(DefaultTimeouts.AvailabilityCheckInterval)
	defer ticker.Stop()

	w.signalAvailability(d)

	for {
		select {
		case <-ctx.Done():
			slog.Info("👷 Worker context canceled, stopping", "worker_id", w.id)
			return
		case <-d.stopCh: //general signal from dispatcher
			slog.Info("👷 Worker received stop signal, stopping", "worker_id", w.id)
			w.metrics.DecrementActiveWorkers()
			return
		case <-w.stopWorkerChan: //signal to terminate worker
			slog.Info("👷 Worker received stop worker signal, stopping", "worker_id", w.id)
			w.metrics.DecrementActiveWorkers()
			return

		//remember to implement a centralized idle ticker system for higher load scenerios
		case <-ticker.C:
			w.signalAvailability(d)

			if w.idleTimeout > 0 && w.GetJobCount() == 0 && w.IsIdleLongEnough() {
				slog.Info("👷 Worker idle for too long, stopping", "worker_id", w.id)

				select {
				case w.idleTerminationCh <- w.id:
					slog.Info("👷 Worker successfully notified dispatcher of idle termination", "worker_id", w.id)
					return
				case <-time.After(DefaultTimeouts.WorkerShutdownTimeout): // Timeout for sending notification
					slog.Warn("⚠️ Worker timeout notifying dispatcher of idle termination. Proceeding with Stop().", "worker_id", w.id)
				case <-ctx.Done(): // Ensure worker respects context cancellation during notification
					slog.Info("Context cancelled while notifying dispatcher of idle termination", "worker_id", w.id)
					return
				}

				d.removeWorkerInternal(w.id, true)
				return
			}

		case job, ok := <-w.JobChannel:
			if !ok {
				slog.Info("👷 Worker JobChannel closed, stopping", "worker_id", w.id)
			}

			err := w.processTask(job, time.Now())
			if err != nil {
				w.metrics.RecordFailure()
				slog.Error("🔴 Worker failed to process task", "worker_id", w.id, "task_id", job.task.ID)
			}

			w.DecrementJobCount()
			w.signalAvailability(d)
		}
	}
}

func (w *Worker) processTask(job Job, startTime time.Time) error {
	slog.Info("👷 Worker processing task", "worker_id", w.id, "task_id", job.task.ID, "priority", job.task.Priority, "name", job.task.Name)
	// time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond) // simulate staggered processing time

	// Simulate failure for ~20% of tasks
	if rand.Float32() < 0.02 {
		return fmt.Errorf("simulated failure for task %d", job.task.ID)
	}

	switch job.task.Priority {
	case 1:
		w.metrics.RecordSuccess()
		slog.Info("🔴 Worker completed HIGH priority task", "worker_id", w.id, "task_id", job.task.ID, "duration", time.Since(startTime))
	case 2:
		w.metrics.RecordSuccess()
		slog.Info("🟠 Worker completed MEDIUM priority task", "worker_id", w.id, "task_id", job.task.ID, "duration", time.Since(startTime))
	default:
		w.metrics.RecordSuccess()
		slog.Info("🟢 Worker completed LOW priority task", "worker_id", w.id, "task_id", job.task.ID, "duration", time.Since(startTime))
	}
	return nil
}

func (w *Worker) IncrementJobCount() {
	count := atomic.AddInt32(&w.jobCount, 1)
	if count == 1 {
		w.metrics.IncrementActiveWorkers()
	}
	w.idleSince.Store(time.Time{}) // Reset idle time
}

func (w *Worker) DecrementJobCount() {
	count := atomic.AddInt32(&w.jobCount, -1)
	if count == 0 {
		w.metrics.DecrementActiveWorkers()
		w.idleSince.Store(time.Now())
	}
}

func (w *Worker) GetJobCount() int32 {
	return atomic.LoadInt32(&w.jobCount)
}

func (w *Worker) IsIdleLongEnough() bool {
	idleTime, ok := w.idleSince.Load().(time.Time)
	if !ok || idleTime.IsZero() {
		return false
	}

	return time.Since(idleTime) >= w.idleTimeout
}
