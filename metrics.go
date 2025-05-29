package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
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

// In metrics.go
func (m *Metrics) Report() string {
	// No need for lock here since we're using atomic operations
	return fmt.Sprintf(
		"Metrics | Jobs in Queue: %d | Active Workers: %d | Successful Jobs: %d | Failed Jobs: %d",
		atomic.LoadInt64(&m.jobsQueueCount),
		atomic.LoadInt64(&m.activeWorkers),
		atomic.LoadInt64(&m.successfulJobs),
		atomic.LoadInt64(&m.failedJobs),
	)
}

// Add a snapshot method for cases where atomic consistency is needed
func (m *Metrics) Snapshot() (jobsQueue, activeWorkers, successful, failed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobsQueue = atomic.LoadInt64(&m.jobsQueueCount)
	activeWorkers = atomic.LoadInt64(&m.activeWorkers)
	successful = atomic.LoadInt64(&m.successfulJobs)
	failed = atomic.LoadInt64(&m.failedJobs)

	return
}

func (m *Metrics) WriteToLog(logDir string) error {
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, fmt.Sprintf("metrics-%s.log", timestamp))

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()

	// Get metrics snapshot
	jobsQueue, activeWorkers, successful, failed := m.Snapshot()

	// Format log entry with timestamp
	logEntry := fmt.Sprintf("[%s] Queue: %d, Workers: %d, Success: %d, Failed: %d\n",
		time.Now().Format("15:04:05"),
		jobsQueue,
		activeWorkers,
		successful,
		failed,
	)

	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}

	return nil
}

// Modify setupMetricsReporting to include file logging
func setupMetricsReporting(ctx context.Context, dispatcher *Dispatcher) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	logDir := "logs"

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Println("📊 " + dispatcher.metrics.Report())

			// Write metrics to log file
			if err := dispatcher.metrics.WriteToLog(logDir); err != nil {
				fmt.Printf("Error writing metrics to log: %v\n", err)
			}
		}
	}
}
