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
	heapSize       int64 
	jobsQueueCount int64 
	activeWorkers  int64 
	successfulJobs int64
	failedJobs     int64
	totalWorkers   int64 
	mu             sync.Mutex
}

// NewMetrics initializes a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Heap size operations
// func (m *Metrics) IncrementHeapSize() {
// 	fmt.Println("🔔 Heap size incremented")
// 	atomic.AddInt64(&m.heapSize, 1)
// }

func (m *Metrics) IncrementHeapSize() {
    atomic.AddInt64(&m.heapSize, 1)
}


func (m *Metrics) DecrementHeapSize() {
	atomic.AddInt64(&m.heapSize, -1)
}

func (m *Metrics) GetHeapSize() int64 {
	return atomic.LoadInt64(&m.heapSize)
}

// Jobs queue operations
func (m *Metrics) IncrementJobsQueueCount() {
	atomic.AddInt64(&m.jobsQueueCount, 1)
}

func (m *Metrics) DecrementJobsQueueCount() {
	atomic.AddInt64(&m.jobsQueueCount, -1)
}

func (m *Metrics) GetJobsQueueCount() int64 {
	return atomic.LoadInt64(&m.jobsQueueCount)
}

// Worker tracking operations
func (m *Metrics) IncrementTotalWorkers() {
	atomic.AddInt64(&m.totalWorkers, 1)
}

func (m *Metrics) DecrementTotalWorkers() {
	atomic.AddInt64(&m.totalWorkers, -1)
}

func (m *Metrics) GetTotalWorkers() int64 {
	return atomic.LoadInt64(&m.totalWorkers)
}

// Active worker operations (workers currently processing jobs)
func (m *Metrics) IncrementActiveWorkers() {
	atomic.AddInt64(&m.activeWorkers, 1)
}

func (m *Metrics) DecrementActiveWorkers() {
	atomic.AddInt64(&m.activeWorkers, -1)
}

func (m *Metrics) GetActiveWorkers() int64 {
	return atomic.LoadInt64(&m.activeWorkers)
}

// Job completion tracking
func (m *Metrics) RecordSuccess() {
	atomic.AddInt64(&m.successfulJobs, 1)
}

func (m *Metrics) RecordFailure() {
	atomic.AddInt64(&m.failedJobs, 1)
}

func (m *Metrics) GetSuccessfulJobs() int64 {
	return atomic.LoadInt64(&m.successfulJobs)
}

func (m *Metrics) GetFailedJobs() int64 {
	return atomic.LoadInt64(&m.failedJobs)
}

// Report generates a formatted metrics string
func (m *Metrics) Report() string {
	return fmt.Sprintf(
		"Heap: %d | Jobs Queue: %d | Total Workers: %d | Active Workers: %d | Successful: %d | Failed: %d",
		atomic.LoadInt64(&m.heapSize),
		atomic.LoadInt64(&m.jobsQueueCount),
		atomic.LoadInt64(&m.totalWorkers),
		atomic.LoadInt64(&m.activeWorkers),
		atomic.LoadInt64(&m.successfulJobs),
		atomic.LoadInt64(&m.failedJobs),
	)
}

func (m *Metrics) Snapshot() (heapSize, jobsQueue, totalWorkers, activeWorkers, successful, failed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	heapSize = atomic.LoadInt64(&m.heapSize)
	jobsQueue = atomic.LoadInt64(&m.jobsQueueCount)
	totalWorkers = atomic.LoadInt64(&m.totalWorkers)
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
	heapSize, jobsQueue, totalWorkers, activeWorkers, successful, failed := m.Snapshot()

	// Format log entry with timestamp
	logEntry := fmt.Sprintf("[%s] Heap: %d, Queue: %d, Total Workers: %d, Active: %d, Success: %d, Failed: %d\n",
		time.Now().Format("15:04:05"),
		heapSize,
		jobsQueue,
		totalWorkers,
		activeWorkers,
		successful,
		failed,
	)

	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}

	return nil
}

// setupMetricsReporting runs in a separate goroutine to report metrics periodically
func setupMetricsReporting(ctx context.Context, dispatcher *Dispatcher) {
	go func() {
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
	}()
}