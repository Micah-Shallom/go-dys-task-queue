package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// dispatcher implements a worker pool pattern
type Dispatcher struct {
	queue      *PriorityJobQueue
	workerPool []*Worker
	numWorkers int
	wg         sync.WaitGroup
	stopCh     chan struct{} //channel to signal all worker to stop
	metrics    *Metrics
	disLock    sync.Mutex
}

func NewDispatcher(numWorkers int) *Dispatcher {
	fmt.Println("🚀 Dispatcher initialized!")
	return &Dispatcher{
		queue:      NewPriorityJobQueue(),
		numWorkers: numWorkers,
		metrics:    NewMetrics(),
		workerPool: make([]*Worker, 0, numWorkers),
		stopCh:     make(chan struct{}),
	}
}

// receiveTasksFromHeap listens for new tasks in the priority queue and dispatches them to the worker pool
func (d *Dispatcher) receiveTasksFromHeap() {
	for {
		<-d.queue.notifyNewTask // Block until a new task is signaled

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
			d.queue.jobsQueue <- job
			d.queue.metrics.IncrementJobsQueueCount()
		}
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	fmt.Println("🟢 Starting dispatcher with worker pool...")

	// Spawn workers
	for i := 0; i < d.numWorkers; i++ {
		worker := NewWorker(i, d.metrics)
		d.workerPool = append(d.workerPool, worker)

		d.wg.Add(1)
		go worker.Start(ctx, d.stopCh, &d.wg, d.queue)
		fmt.Printf("👷 Worker %d added to the pool\n", i)
	}

	go d.receiveTasksFromHeap()

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
			go func(job Job) {
				worker := d.findAvailableWorker()
				if worker != nil {

					select {
					case worker.JobChannel <- job:
						fmt.Printf("📤 Job %d dispatched to worker %d\n", job.task.ID, worker.id)
					case <- time.After(500 * time.Millisecond):
						//handle timeout 
						fmt.Printf("⚠️ Job %d dispatch to worker %d timed out\n", job.task.ID, worker.id)
						d.queue.PushToHeap(job.task) // Requeue the job
					}
				} else {
					fmt.Println("⚠️ No available workers to process the job")
					d.queue.PushToHeap(job.task) // Requeue the job
				}
			}(job)
		}
	}
}

func (d *Dispatcher) findAvailableWorker() *Worker {
	for _, worker := range d.workerPool {
		if !worker.running {
			return worker
		}
	}
	return nil
}

func (d *Dispatcher) Stop() {
	fmt.Println("🔴 Shutting down dispatcher...")
	d.stopCh <- struct{}{}
	fmt.Println("⏳ Waiting for all workers to complete...")
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
	fmt.Println("✅ All workers have completed, dispatcher shutdown complete!")
}
