package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// dispatcher implements a worker pool pattern
type Dispatcher struct {
	queue             *PriorityJobQueue //jobs are read from this queue into the heap for priority processing
	stopCh            chan struct{}     //channel to signal all worker to stop
	idleTerminationCh chan int          //channel to signal worker termination when idle
	workers           map[int]*Worker   // Map of worker ID to Worker
	numWorkers        int
	wg                sync.WaitGroup
	disLock           sync.Mutex
	setMU             sync.RWMutex
	metrics           *Metrics
	nextWorkerID      int
	availableSet      map[int]bool // Set of available worker IDs
	availableWorkers  chan int
}

func NewDispatcher(numWorkers int) *Dispatcher {
	slog.Info("🚀 Dispatcher initialized", "num workers", numWorkers)
	return &Dispatcher{
		queue:             NewPriorityJobQueue(),
		numWorkers:        numWorkers,
		metrics:           NewMetrics(),
		availableWorkers:  make(chan int, numWorkers), // Buffered channel to hold available workers
		availableSet:      make(map[int]bool),         // Set to track available workers
		workers:           make(map[int]*Worker),
		stopCh:            make(chan struct{}),
		idleTerminationCh: make(chan int, numWorkers),
	}
}

func (d *Dispatcher) GetWorkerByID(workerID int) (*Worker, bool) {
	d.disLock.Lock()
	defer d.disLock.Unlock()
	worker, exists := d.workers[workerID]
	return worker, exists
}

func (d *Dispatcher) GetAllWorkers() map[int]*Worker {
	d.disLock.Lock()
	defer d.disLock.Unlock()

	workers := make(map[int]*Worker)
	for id, worker := range d.workers {
		workers[id] = worker
	}
	return workers
}

func (d *Dispatcher) AddWorker(ctx context.Context) int {
	d.disLock.Lock()
	defer d.disLock.Unlock()

	workerID := d.nextWorkerID
	d.nextWorkerID++

	worker := NewWorker(workerID, d.metrics, 5000, d)
	d.workers[workerID] = worker
	d.metrics.IncrementTotalWorkers()

	d.wg.Add(1)
	go worker.Start(ctx, d)

	slog.Info("🧑‍💻 Worker added to the pool", "worker_id", workerID)
	return workerID
}

func (d *Dispatcher) RemoveWorkerByID(workerID int) error {
	_, err := d.removeWorkerInternal(workerID, true) // lockNeeded is true
	if err != nil {
		slog.Error("Error removing worker by ID", "worker_id", workerID, "error", err)
		return err
	}
	slog.Info("👷 Worker removed from the pool by explicit request", "worker_id", workerID)
	return nil
}

// receiveTasksFromHeap listens for new tasks in the priority queue and dispatches them to the worker pool
func (d *Dispatcher) receiveTasksFromHeap(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("🛑 Dispatcher context canceled, stopping task reception")
			return
		case <-d.queue.notifyNewTask: // Block until a new task is signaled
			for {
				d.queue.mu.Lock()
				if d.queue.taskHeap.Len() == 0 {
					d.queue.mu.Unlock()
					break // Exit inner loop if no more tasks
				}

				task := d.queue.taskHeap.Pop()
				d.metrics.DecrementHeapSize()
				d.queue.mu.Unlock()

				job := Job{task: task.(Task)}

				select {
				case d.queue.jobsQueue <- job:
					d.metrics.IncrementJobsQueueCount()
				default:
					slog.Warn("⚠️ jobsQueue is full, skipping task delivery")
					err := d.queue.PushToHeap(task.(Task), d) // requeue the task
					if err != nil {
						slog.Error("❗ Error re-queuing task", "task_id", task.(Task).ID, "err", err)
					} else {
						slog.Info("🔄 Task re-queued to the heap", "task_id", task.(Task).ID)
					}
					break
				}
			}
		}
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	slog.Info("🏁 Starting dispatcher with worker pool...")

	// Spawn workers
	for i := range d.numWorkers {
		_ = i
		d.AddWorker(ctx)
	}

	go d.receiveTasksFromHeap(ctx)

	go d.dispatch(ctx)
	go d.handleIdleTermination(ctx)

	go func() {
		<-ctx.Done()
		d.StopDispatch()
	}()

	slog.Info("✅ Worker pool ready", "num_workers", d.numWorkers)
}

func (d *Dispatcher) handleIdleTermination(ctx context.Context) {
	slog.Info("🕰️ Starting idle termination handler...")
	defer slog.Info("🛑 Idle termination handler stopped")

	for {
		select {
		case <-ctx.Done():
			slog.Info("🛑 Idle termination context canceled, stopping handler")
			return
		case workerID := <-d.idleTerminationCh:
			slog.Info("🗑️ Worker idle for too long, terminating", "worker_id", workerID)

			removedWorker, err := d.removeWorkerInternal(workerID, true)
			if err != nil {
				slog.Warn("❗ Error removing worker", "worker_id", workerID, "err", err)
				continue
			}

			slog.Info("🗑️ Worker removed from the pool", "worker_id", removedWorker.id)
		}
	}
}

func (d *Dispatcher) removeWorkerInternal(workerID int, lock bool) (*Worker, error) {
	if lock {
		d.disLock.Lock()
		defer d.disLock.Unlock()
	}

	worker, exists := d.workers[workerID]
	if !exists {
		return nil, fmt.Errorf("worker %d not found for internal removal", workerID)
	}

	// Clean up worker from available set
	d.setMU.Lock()
	delete(d.availableSet, workerID)
	d.setMU.Unlock()

	// Stop the worker and clean up
	worker.Stop(d)
	delete(d.workers, workerID)
	d.metrics.DecrementTotalWorkers()

	return worker, nil
}

func (d *Dispatcher) dispatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("🛑 Dispatcher context canceled, stopping dispatching")
			return

		case job := <-d.queue.jobsQueue:

			d.metrics.DecrementJobsQueueCount()
			workerID := d.findAvailableWorker()
			if workerID == -1 {
				slog.Warn("⚠️ No available workers to process the job")
				err := d.queue.PushToHeap(job.task, d) // Requeue the job
				if err != nil {
					slog.Error("❗ Error re-queuing job", "job_id", job.task.ID, "err", err)
				}
				slog.Info("🔄 Job re-queued to the heap", "job_id", job.task.ID)
				continue
			}

			worker, exists := d.GetWorkerByID(workerID)
			if !exists {
				slog.Error("❌ Worker not found, re-queuing job", "worker_id", workerID, "job_id", job.task.ID)
				err := d.queue.PushToHeap(job.task, d)
				if err != nil {
					slog.Error("❗ Error re-queuing job", "job_id", job.task.ID, "err", err)
				}
				continue
			}

			go func(job Job, w *Worker, wID int) {
				select {
				case worker.JobChannel <- job:
					w.IncrementJobCount()
				case <-time.After(DefaultTimeouts.TaskDispatchTimeout):
					//handle timeout
					slog.Warn("⏰ Job dispatch to worker timed out", "job_id", job.task.ID, "worker_id", worker.id)
					err := d.queue.PushToHeap(job.task, d) // Requeue the job
					if err != nil {
						slog.Error("❗ Error re-queuing job", "job_id", job.task.ID, "err", err)
					} else {
						slog.Info("🔄 Job re-queued to the heap", "job_id", job.task.ID)
					}
				}
			}(job, worker, workerID)
		}
	}
}

func (d *Dispatcher) findAvailableWorker() int {
	select {
	case workerID := <-d.availableWorkers:
		d.setMU.Lock()
		delete(d.availableSet, workerID)
		d.setMU.Unlock()
		return workerID
	default:
		return -1 // No available workers
	}
}

func (d *Dispatcher) StopDispatch() {
	slog.Info("🛑 Shutting down dispatcher...")
	close(d.stopCh)

	slog.Info("⏳ Waiting for all workers to complete...")
}

func (d *Dispatcher) StopWorker(worker *Worker) error {
	return d.RemoveWorkerByID(worker.id)
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
	slog.Info("🎉 All workers have completed, dispatcher shutdown complete!")
}
