package main

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"
)

// dispatcher implements a worker pool pattern
type Dispatcher struct {
	queue            *PriorityJobQueue //jobs are read from this queue into the heap for priority processing
	workerPool       []*Worker
	availableWorkers chan *Worker
	numWorkers       int
	wg               sync.WaitGroup
	stopCh           chan struct{} //channel to signal all worker to stop
	metrics          *Metrics
	disLock          sync.Mutex
}

func NewDispatcher(numWorkers int) *Dispatcher {
	fmt.Println("🚀 Dispatcher initialized!")
	return &Dispatcher{
		queue:            NewPriorityJobQueue(),
		numWorkers:       numWorkers,
		metrics:          NewMetrics(),
		workerPool:       make([]*Worker, 0, numWorkers),
		availableWorkers: make(chan *Worker, numWorkers),
		stopCh:           make(chan struct{}), //general stopChan to signal all workers to stop
	}
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
		worker := NewWorker(i, d.metrics, 5)
		d.workerPool = append(d.workerPool, worker)

		d.wg.Add(1)
		go worker.Start(ctx, d)
		fmt.Printf("👷 Worker %d added to the pool\n", i)
	}

	go d.receiveTasksFromHeap(ctx)

	go d.dispatch(ctx)

	go func() {
		<-ctx.Done()
		d.Stop()
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
			worker := d.findAvailableWorker()
			if worker == nil {
				fmt.Println("⚠️ No available workers to process the job")
				err := d.queue.PushToHeap(job.task) // Requeue the job
				if err != nil {
					fmt.Printf("❌ Error re-queuing job %d: %v\n", job.task.ID, err)
				}
				continue
			}

			go func(job Job, w *Worker) {
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
			}(job, worker)
		}
	}
}

func (d *Dispatcher) findAvailableWorker() *Worker {
	select {
	case worker := <-d.availableWorkers:
		return worker
	default:
		return nil
	}
}

func (d *Dispatcher) Stop() {
	fmt.Println("🔴 Shutting down dispatcher...")
	close(d.stopCh)

	fmt.Println("⏳ Waiting for all workers to complete...")
}

func (d *Dispatcher) StopWorker(worker *Worker) error {
	fmt.Printf("👷 Worker %d: Stopping\n", worker.id)
	worker.Stop()

	d.disLock.Lock()
	for i, w := range d.workerPool {
		if w.id == worker.id {
			d.workerPool = slices.Delete(d.workerPool, i, i+1)
			break
		}
	}
	d.disLock.Unlock()
	d.wg.Done()
	return nil
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
	fmt.Println("✅ All workers have completed, dispatcher shutdown complete!")
}
