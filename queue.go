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
	jobsQueue     chan Job     //contains jobs to be processed
	mu            sync.Mutex
	notifyNewTask chan struct{}
}

func NewPriorityJobQueue() *PriorityJobQueue {
	pq := &PriorityJobQueue{
		jobsQueue:     make(chan Job, 1000),
		notifyNewTask: make(chan struct{}, 1),
		taskHeap: priorityHeap{
			tasks:  make([]Task, 0),
			closed: false,
		},
	}

	heap.Init(&pq.taskHeap)

	return pq
}

func (pq *PriorityJobQueue) PushToHeap(task Task, d *Dispatcher) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.taskHeap.closed {
		return fmt.Errorf("task heap is closed")
	}

	heap.Push(&pq.taskHeap, task)
	d.metrics.IncrementHeapSize()

	//signal the worker pool that a new task is available
	select {
	case pq.notifyNewTask <- struct{}{}:
	default:
		// non-blocking send
	}

	return nil
}

func (pq *PriorityJobQueue) PopFromJobQueue(d *Dispatcher) (Job, bool) {
	task, ok := <-pq.jobsQueue
	if ok {
		d.metrics.DecrementJobsQueueCount()
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

func (h *priorityHeap) Push(x any) {
	h.tasks = append(h.tasks, x.(Task))
}

func (h *priorityHeap) Pop() any {
	old := h.tasks
	n := len(old)
	x := old[n-1]
	h.tasks = old[0 : n-1]
	return x
}
