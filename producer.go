package main

import (
	"fmt"
	"math/rand"
	"time"
)

func generateData() []Task {
	var inMemoryDatabase []Task

	for i := 0; i < 1_000; i++ {
		var priority int
		r := rand.Intn(10)
		switch {
		case r < 2:
			priority = 1
		case r < 5:
			priority = 2
		default:
			priority = 3
		}
		task := Task{ID: i, Priority: priority, Name: fmt.Sprintf("Task-%d", i)}
		inMemoryDatabase = append(inMemoryDatabase, task)
	}
	return inMemoryDatabase
}

func Producer(data []Task, d *Dispatcher) {

	for {
		d.disLock.Lock()

		if len(data) > 0 {
			task := data[0]
			d.queue.PushToHeap(task)
			data = data[1:]
			d.disLock.Unlock()
		} else {
			d.disLock.Unlock()

			time.Sleep(100 * time.Millisecond)
		}
	}
}
