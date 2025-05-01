package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Metrics tracks task queue statistics.
type Metrics struct {
	heapSize       int64 // Size of the priority heap (taskqueue)
	jobsQueueCount int64 // Number of jobs in the queue (jobqueue)
	activeWorkers  int64 // Number of workers currently processing tasks
	successfulJobs int64 // Number of successfully completed jobs
	failedJobs     int64 // Number of failed jobs
	mu             sync.Mutex
}


// NewMetrics initializes a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// IncrementHeapSize increments the heap size.
func (m *Metrics) IncrementHeapSize() {
	atomic.AddInt64(&m.heapSize, 1)
}

// DecrementHeapSize decrements the heap size.
func (m *Metrics) DecrementHeapSize() {
	atomic.AddInt64(&m.heapSize, -1)
}

// IncrementJobsQueueCount increments the jobs queue count.
func (m *Metrics) IncrementJobsQueueCount() {
	atomic.AddInt64(&m.jobsQueueCount, 1)
}

// DecrementJobsQueueCount decrements the jobs queue count.
func (m *Metrics) DecrementJobsQueueCount() {
	atomic.AddInt64(&m.jobsQueueCount, -1)
}

// IncrementActiveWorkers increments the active workers count.
func (m *Metrics) IncrementActiveWorkers() {
	atomic.AddInt64(&m.activeWorkers, 1)
}

// DecrementActiveWorkers decrements the active workers count.
func (m *Metrics) DecrementActiveWorkers() {
	atomic.AddInt64(&m.activeWorkers, -1)
}

// RecordSuccess increments the successful jobs count.
func (m *Metrics) RecordSuccess() {
	atomic.AddInt64(&m.successfulJobs, 1)
}

// RecordFailure increments the failed jobs count.
func (m *Metrics) RecordFailure() {
	atomic.AddInt64(&m.failedJobs, 1)
}

// Report returns a formatted string of all metrics.
func (m *Metrics) Report() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf(
		"Metrics | Jobs in Queue: %d | Active Workers: %d | Successful Jobs: %d | Failed Jobs: %d",
		atomic.LoadInt64(&m.jobsQueueCount),
		atomic.LoadInt64(&m.activeWorkers),
		atomic.LoadInt64(&m.successfulJobs),
		atomic.LoadInt64(&m.failedJobs),
	)
}
