package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var (
	totalTasksGenerated atomic.Int64
	closeTaskStream     sync.Once
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
			currentCount := totalTasksGenerated.Load()
			if currentCount > 1_000_000_000 {
				closeTaskStream.Do(func() {
					slog.Info("🎯 Reached 1 billion tasks, stopping task production", "total_tasks", currentCount-1)
					close(taskstream)
				})
				return
			}

			priority := randomPriority()
			task := Task{
				ID:       id,
				Priority: priority,
				Name:     fmt.Sprintf("Task-%d", id),
			}

			metrics.IncrementTotalTasks()
			select {
			case taskstream <- task:
				totalTasksGenerated.Add(1)
				id++
			default:
				slog.Info("🛑 TaskFeeder queue is full, sleeping for 500ms")
				time.Sleep(time.Duration(rand.Intn(3)) * time.Second)
			}
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
