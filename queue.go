package main

import (
	"container/heap"
	"sync"
)

type Task struct {
	ID       int
	Priority int // 1=high, 2=medium, 3=low
	Name     string
}

type Job struct {
	task Task
}

type PriorityJobQueue struct {
	taskHeap priorityHeap //handles priority scheduling
	jobqueue chan Job     //
	mu       sync.Mutex
}

func NewPriorityJobQueue() *PriorityJobQueue {
	pq := &PriorityJobQueue{
		jobqueue: make(chan Job, 1),
	}

	go pq.process()

	return pq
}

func (pq *PriorityJobQueue) Push(task Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	heap.Push(&pq.taskHeap, task)
}

func (pq *PriorityJobQueue) process() {
	for {
		pq.mu.Lock()
		defer pq.mu.Unlock()

		if pq.taskHeap.Len() > 0 {
			task := pq.taskHeap.Pop()
			pq.jobqueue <- Job{task: task.(Task)}
		}
	}
}

func (pq *PriorityJobQueue) Pop() (Job, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	task, ok := <-pq.jobqueue
	return Job{task: task.task}, ok
}

// implement the heap interface
type priorityHeap struct {
	tasks  []Task
	closed bool
}

func (h *priorityHeap) Len() int           { return len(h.tasks) }
func (h *priorityHeap) Less(i, j int) bool { return h.tasks[i].Priority < h.tasks[j].Priority }
func (h *priorityHeap) Swap(i, j int)      { h.tasks[i], h.tasks[j] = h.tasks[j], h.tasks[i] }

func (h *priorityHeap) Push(x interface{}) {
	h.tasks = append(h.tasks, x.(Task))
}

func (h *priorityHeap) Pop() interface{} {
	old := h.tasks
	n := len(old)
	x := old[n-1]
	h.tasks = old[0 : n-1]
	return x
}
