package main

import (
	"context"
	"fmt"
	"sync"
)

// dispatcher implements a worker pool pattern
type Dispatcher struct {
	queue      *PriorityJobQueue
	workerPool []*Worker
	numWorkers int
	wg         sync.WaitGroup
	stopCh     chan struct{} //channel to signal all worker to stop
	metrics    *Metrics
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

func (d *Dispatcher) receiveTasks() {
	//pop from the heap and populate the job queue
	for {
		if d.queue.taskHeap.Len() > 0 {
			task := d.queue.taskHeap.Pop()
			job := Job{task: task.(Task)}
			d.queue.jobqueue <- job
		}
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	fmt.Println("🟢 Starting dispatcher with worker pool...")

	for i := range d.numWorkers {
		worker := NewWorker(i, d.queue, d.metrics)
		d.workerPool = append(d.workerPool, worker)

		d.wg.Add(1)
		go worker.Start(ctx, d.stopCh, &d.wg)
		fmt.Printf("👷 Worker %d added to the pool\n", i)

		//start receiving tasks
		go d.receiveTasks()
		//start dispatching tasks
		go d.dispatch()

		//monitor for context cancellation
		go func() {
			<-ctx.Done()
			d.Stop()
		}()
	}

	fmt.Printf("✅ Worker pool ready with %d workers\n", d.numWorkers)
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <- d.queue.jobqueue:
			go func(job Job) {
				worker := d.findAvailableWorker()
				if worker != nil {
					worker.JobChannel <- job
					fmt.Printf("📤 Job %d dispatched to worker %d\n", job.task.ID, worker.id)
				} else {
					fmt.Println("⚠️ No available workers to process the job")
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
	close(d.stopCh) // Signal all workers to stop
	fmt.Println("⏳ Waiting for all workers to complete...")
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
	fmt.Println("✅ All workers have completed, dispatcher shutdown complete!")
}

func (d *Dispatcher) Submit(task Task) {
	fmt.Println("📥 Task submitted to the queue!")
	d.queue.Enqueue(task)
}
