package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Metrics struct {
	successfulTasks int32
	failedTasks     int32
	mu              sync.Mutex
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) RecordSuccess() {
	atomic.AddInt32(&m.successfulTasks, 1)
}

func (m *Metrics) RecordFailure() {
	atomic.AddInt32(&m.failedTasks, 1)
}

func (m *Metrics) Report() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fmt.Sprintf("Metrics | Successful: %d | Failed: %d", m.successfulTasks, m.failedTasks)
}
