package main

import "sync"

type Task struct {
	ID       int
	Priority int // 1=high, 2=medium, 3=low
	Name     string
}

type PriorityQueue struct {
	tasks []Task
	mu    sync.Mutex
	cond  *sync.Cond
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{tasks: make([]Task, 0)}
	pq.cond = sync.NewCond(&pq.mu)
	return pq
}

func (pq *PriorityQueue) Enqueue(task Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	for i := len(pq.tasks) - 1; i > 0 && pq.tasks[i].Priority < pq.tasks[i-1].Priority; i-- {
		pq.tasks[i], pq.tasks[i-1] = pq.tasks[i-1], pq.tasks[i]
	}
	pq.cond.Signal()
}

func (pq *PriorityQueue) Dequeue() Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	for len(pq.tasks) == 0 {
		pq.cond.Wait() //blocks until signal is sent that a task has been enqueued
	}
	task := pq.tasks[0]
	pq.tasks = pq.tasks[1:]
	return task
}

func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.tasks)
}
