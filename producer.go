package main

import (
	"context"
	"fmt"
	"log/slog"
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

func TaskFeeder(ctx context.Context, taskstream chan<- Task, metrics *Metrics) {
	id := 0
	for {
		select {
		case <-ctx.Done():
			slog.Info("🛑 TaskFeeder shutting down")
			close(taskstream)
			return
		default:
			priority := randomPriority()
			task := Task{
				ID:       id,
				Priority: priority,
				Name:     fmt.Sprintf("Task-%d", id),
			}

			metrics.IncrementTotalTasks()
			taskstream <- task
			id++
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond) // simulate staggered arrival
		}
	}
}

func Producer(ctx context.Context, taskStream <-chan Task, d *Dispatcher) {
	slog.Info("👂 Starting Producer to listen on taskStream and push to dispatcher's heap...")
	defer slog.Info("🛑 Producer shutting down.")
	for {
		select {
		case <-ctx.Done():
			slog.Info("🛑 Producer shutting down")
			return
		case task, ok := <-taskStream:
			if !ok {
				slog.Info("🛑 Task stream closed, producer exiting")
				return
			}
			d.queue.PushToHeap(task, d)
		}
	}
}
