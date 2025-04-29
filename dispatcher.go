package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Dispatcher struct {
	queue         *PriorityQueue
	workers       []*Worker
	maxWorkers    int
	scalingFactor int
	wg            sync.WaitGroup
	metrics       *Metrics
}

func NewDispatcher(maxWorkers, scalingFactor int) *Dispatcher {
	fmt.Println("🚀 Dispatcher initialized!")
	return &Dispatcher{
		queue:         NewPriorityQueue(),
		maxWorkers:    maxWorkers,
		scalingFactor: scalingFactor,
		metrics:       NewMetrics(),
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	fmt.Println("🟢 Dispatcher started!")
	d.wg.Add(1)
	go d.scaleWorkers(ctx)
}

func (d *Dispatcher) Shutdown() {
	fmt.Println("🔴 Shutting down dispatcher...")
	for _, worker := range d.workers {
		fmt.Printf("🛑 Stopping worker %d...\n", worker.id)
		worker.Stop()
	}
	d.workers = nil
	fmt.Println("✅ Dispatcher shutdown complete!")
}

func (d *Dispatcher) scaleWorkers(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("⚠️ Context canceled, shutting down workers...")
			d.Shutdown()
			return

		default:
			queueLen := d.queue.Len()
			desiredWorkers := queueLen / d.scalingFactor
			if desiredWorkers < 1 {
				desiredWorkers = 1
			}
			if desiredWorkers > d.maxWorkers {
				desiredWorkers = d.maxWorkers
			}

			fmt.Printf("📊 Queue length: %d, Desired workers: %d, Current workers: %d\n", queueLen, desiredWorkers, len(d.workers))

			// Scaling up workers
			for len(d.workers) < desiredWorkers {
				worker := NewWorker(len(d.workers)+1, d.queue, d.metrics)
				d.workers = append(d.workers, worker)
				fmt.Printf("⬆️ Starting worker %d...\n", worker.id)
				d.wg.Add(1)
				go worker.Start(ctx, &d.wg)
			}

			// Scaling down workers
			for len(d.workers) > desiredWorkers {
				if len(d.workers) == 0 {
					break
				}
				worker := d.workers[len(d.workers)-1]
				d.workers = d.workers[:len(d.workers)-1]
				fmt.Printf("⬇️ Stopping worker %d...\n", worker.id)
				worker.Stop()
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (d *Dispatcher) Submit(task Task) {
	fmt.Println("📥 Task submitted to the queue!")
	d.queue.Enqueue(task)
}