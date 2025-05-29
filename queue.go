package main

import (
	"container/heap"
	"fmt"
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
	taskHeap      priorityHeap //handles priority scheduling
	jobsQueue     chan Job     //
	mu            sync.Mutex
	notifyNewTask chan struct{}
	metrics       *Metrics
}

func NewPriorityJobQueue() *PriorityJobQueue {
	pq := &PriorityJobQueue{
		jobsQueue: make(chan Job, 1000),
		metrics:   NewMetrics(),
		notifyNewTask: make(chan struct{}, 1), // buffered channel to avoid blocking
		taskHeap: priorityHeap{
			tasks: make([]Task, 0),
			closed: false,
		},
	}

	heap.Init(&pq.taskHeap)

	return pq
}

func (pq *PriorityJobQueue) PushToHeap(task Task) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.taskHeap.closed {
		return fmt.Errorf("task heap is closed")
	}

	heap.Push(&pq.taskHeap, task)

	//signal the worker pool that a new task is available
	select {
	case pq.notifyNewTask <- struct{}{}:
	default:
		// non-blocking send
	}

	pq.metrics.IncrementHeapSize()
	return nil
}

func (pq *PriorityJobQueue) PopFromJobQueue() (Job, bool) {
	task, ok := <-pq.jobsQueue
	if ok {
		pq.metrics.DecrementJobsQueueCount()
	}
	return Job{task: task.task}, ok
}

func (pq *PriorityJobQueue) Close() {
    pq.mu.Lock()
    defer pq.mu.Unlock()

    // Mark the task heap as closed
    pq.taskHeap.closed = true

    // Close channels
    close(pq.jobsQueue)
    close(pq.notifyNewTask)

    // Clear the task heap
    pq.taskHeap.tasks = nil
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


















