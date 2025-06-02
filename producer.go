package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

func randomPriority() int {
	r := rand.Intn(10)
	switch {
	case r < 2:
		return 1
	case r < 5:
		return 2
	default:
		return 3
	}
}

func TaskFeeder(ctx context.Context, taskstream chan<- Task) {
	id := 0
	for {
		select {
		case <-ctx.Done():
			fmt.Println("🛑 TaskFeeder shutting down")
			close(taskstream)
		default:
			priority := randomPriority()
			task := Task{
				ID:       id,
				Priority: priority,
				Name:     fmt.Sprintf("Task-%d", id),
			}
			taskstream <- task
			id++
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond) // simulate staggered arrival
		}
	}
}

func Producer(ctx context.Context, taskStream <-chan Task, d *Dispatcher) {

	for {
		select {
		case <-ctx.Done():
			fmt.Println("🛑 Producer shutting down")
			return
		case task, ok := <-taskStream:
			if !ok {
				fmt.Println("🛑 Task stream closed, producer exiting")
				return
			}
			d.queue.PushToHeap(task, d)
		}
	}
}
