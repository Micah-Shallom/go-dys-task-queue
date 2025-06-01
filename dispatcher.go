package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// dispatcher implements a worker pool pattern
type Dispatcher struct {
	queue            *PriorityJobQueue //jobs are read from this queue into the heap for priority processing
	stopCh           chan struct{}     //channel to signal all worker to stop
	availableWorkers chan int
	workers          map[int]*Worker // Map of worker ID to Worker
	numWorkers       int
	wg               sync.WaitGroup
	disLock          sync.Mutex
	metrics          *Metrics
	nextWorkerID     int
}

func NewDispatcher(numWorkers int) *Dispatcher {
	fmt.Println("🚀 Dispatcher initialized!")
	return &Dispatcher{
		queue:            NewPriorityJobQueue(),
		numWorkers:       numWorkers,
		metrics:          NewMetrics(),
		availableWorkers: make(chan int, numWorkers), // Buffered channel to hold available workers
		workers:          make(map[int]*Worker),
		stopCh:           make(chan struct{}), //general stopChan to signal all workers to stop
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

	worker := NewWorker(workerID, d.metrics, 5)
	d.workers[workerID] = worker

	d.wg.Add(1)
	go worker.Start(ctx, d)

	fmt.Printf("👷 Worker %d added to the pool\n", workerID)
	return workerID
}

func (d *Dispatcher) RemoveWorker(workerID int) error {
	d.disLock.Lock()
	defer d.disLock.Unlock()

	worker, exists := d.workers[workerID]
	if !exists {
		return fmt.Errorf("worker %d not found", workerID)
	}

	worker.Stop()
	delete(d.workers, workerID)

	return nil
}

// receiveTasksFromHeap listens for new tasks in the priority queue and dispatches them to the worker pool
func (d *Dispatcher) receiveTasksFromHeap(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("🔴 Dispatcher context canceled, stopping task reception")
			return
		case <-d.queue.notifyNewTask: // Block until a new task is signaled
			for {
				d.queue.mu.Lock()
				if d.queue.taskHeap.Len() == 0 {
					d.queue.mu.Unlock()
					break // Exit inner loop if no more tasks
				}

				task := d.queue.taskHeap.Pop()
				d.queue.metrics.DecrementHeapSize()
				d.queue.mu.Unlock()

				job := Job{task: task.(Task)}

				select {
				case d.queue.jobsQueue <- job:
					d.queue.metrics.IncrementJobsQueueCount()
				default:
					fmt.Println("⚠️ jobsQueue is full, skipping task delivery")
					err := d.queue.PushToHeap(task.(Task))
					if err != nil {
						fmt.Printf("❌ Error re-queuing task %d: %v\n", task.(Task).ID, err)
					} else {
						fmt.Printf("🔄 Task %d re-queued to the heap\n", task.(Task).ID)
					}
					break
				}
			}
		}
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	fmt.Println("🟢 Starting dispatcher with worker pool...")

	// Spawn workers
	for i := range d.numWorkers {
		_ = i
		d.AddWorker(ctx)
	}

	go d.receiveTasksFromHeap(ctx)

	go d.dispatch(ctx)

	go func() {
		<-ctx.Done()
		d.StopDispatch()
	}()

	fmt.Printf("✅ Worker pool ready with %d workers\n", d.numWorkers)
}

func (d *Dispatcher) dispatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("🔴 Dispatcher context canceled, stopping dispatching")
			return
		case job := <-d.queue.jobsQueue:
			workerID := d.findAvailableWorker()
			if workerID == -1 {
				fmt.Println("⚠️ No available workers to process the job")
				err := d.queue.PushToHeap(job.task) // Requeue the job
				if err != nil {
					fmt.Printf("❌ Error re-queuing job %d: %v\n", job.task.ID, err)
				}
				continue
			}

			worker, exists := d.GetWorkerByID(workerID)
			if !exists {
				fmt.Printf("❌ Worker %d not found, re-queuing job %d\n", workerID, job.task.ID)
				err := d.queue.PushToHeap(job.task)
				if err != nil {
					fmt.Printf("❌ Error re-queuing job %d: %v\n", job.task.ID, err)
				}
				continue
			}

			go func(job Job, w *Worker, wID int) {
				select {
				case worker.JobChannel <- job:
					fmt.Printf("📤 Job %d dispatched to worker %d\n", job.task.ID, worker.id)
					w.IncrementJobCount()
				case <-time.After(500 * time.Millisecond):
					//handle timeout
					fmt.Printf("⚠️ Job %d dispatch to worker %d timed out\n", job.task.ID, worker.id)
					err := d.queue.PushToHeap(job.task) // Requeue the job
					if err != nil {
						fmt.Printf("❌ Error re-queuing job %d: %v\n", job.task.ID, err)
					} else {
						fmt.Printf("🔄 Job %d re-queued to the heap\n", job.task.ID)
					}
				}
			}(job, worker, workerID)
		}
	}
}

func (d *Dispatcher) findAvailableWorker() int {
	select {
	case workerID := <-d.availableWorkers:
		return workerID
	default:
		return -1 // No available workers
	}
}

func (d *Dispatcher) StopDispatch() {
	fmt.Println("🔴 Shutting down dispatcher...")
	close(d.stopCh)

	fmt.Println("⏳ Waiting for all workers to complete...")
}

func (d *Dispatcher) RemoveWorkerByID(workerID int) error {
	d.disLock.Lock()
	defer d.disLock.Unlock()

	worker, exists := d.workers[workerID]
	if !exists {
		return fmt.Errorf("worker %d not found", workerID)
	}

	worker.Stop()
	delete(d.workers, workerID)

	fmt.Printf("👷 Worker %d removed from the pool\n", workerID)
	return nil
}


func (d *Dispatcher) StopWorker(worker *Worker) error {
	return d.RemoveWorkerByID(worker.id)
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
	fmt.Println("✅ All workers have completed, dispatcher shutdown complete!")
}
